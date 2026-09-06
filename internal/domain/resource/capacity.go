package resource

// P5-W1 容量管理(对标 NRM 基线 C6):端口使用率聚合 + >=80% 阈值预警(复用告警体系)。
// 口径(任务书):使用率 = USED/(USED+IDLE),百分比两位小数;DISABLED/RESERVED 不入分母。
// 对象 = resources 行(type=OLT/SPLITTER);预警级别 WARNING,见 docs/contract/fields.md §4.2.2。

import (
	"context"
	"fmt"
)

// CapacityWarnThresholdPct 容量预警阈值(%):使用率 >= 阈值的对象产生 WARNING 容量告警。
const CapacityWarnThresholdPct = 80.0

// 容量聚合参数白名单:dim 过滤设备类型;order 缺省 usageDesc(任务书:按使用率倒序)。
const (
	DimOLT         = "OLT"
	DimSplitter    = "SPLITTER"
	OrderUsageDesc = "usageDesc"
	OrderUsageAsc  = "usageAsc"
)

// CapacityStat 单设备(OLT/分光器)端口容量统计行。
type CapacityStat struct {
	ResourceID int64   `json:"resourceId"`
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Type       string  `json:"type"` // OLT/SPLITTER
	TotalPorts int64   `json:"totalPorts"`
	UsedPorts  int64   `json:"usedPorts"`
	UsageRate  float64 `json:"usageRate"` // USED/(USED+IDLE) 百分比,两位小数
}

// CapacityScanResult 一轮阈值扫描计数(周期重跑幂等:状态未变化则 Created/Resolved=0)。
type CapacityScanResult struct {
	Scanned  int `json:"scanned"`
	Created  int `json:"created"`
	Resolved int `json:"resolved"`
}

// CapacityAlarmSink 容量预警写告警的最小依赖口(app 装配层适配 device.AlarmService;
// 资源域不跨域 import 告警实现,依赖倒置见 docs/ADR-001)。
type CapacityAlarmSink interface {
	// HasOpenCapacityAlarm 对象是否存在 OPEN 态容量告警(判重依据:告警行即阈值状态)。
	HasOpenCapacityAlarm(ctx context.Context, resourceID int64) (bool, error)
	// CreateCapacityAlarm 对象越限产生一条 WARNING 容量告警。
	CreateCapacityAlarm(ctx context.Context, resourceID int64, content string) error
	// CloseCapacityAlarms 对象回落阈值下,关闭其全部 OPEN 容量告警,返回关闭条数。
	CloseCapacityAlarms(ctx context.Context, resourceID int64) (int64, error)
}

// CapacityService 容量视图聚合 + 阈值预警扫描(admin /resources/capacity* 取数口)。
type CapacityService interface {
	Capacity(ctx context.Context, dim, order string) ([]CapacityStat, error)
	CapacityAlertScan(ctx context.Context, thresholdPct float64, sink CapacityAlarmSink) (CapacityScanResult, error)
}

var _ CapacityService = (*PGStore)(nil)

// Capacity 按维度聚合端口使用率;dim 空=全部类型;order 仅接受 usageDesc/usageAsc。
func (s *PGStore) Capacity(ctx context.Context, dim, order string) ([]CapacityStat, error) {
	switch dim {
	case "", DimOLT, DimSplitter:
	default:
		return nil, fmt.Errorf("resource: invalid dim %q", dim)
	}
	dir := "DESC"
	switch order {
	case "", OrderUsageDesc:
	case OrderUsageAsc:
		dir = "ASC"
	default:
		return nil, fmt.Errorf("resource: invalid order %q", order)
	}
	rows, err := s.db.Query(ctx, `
		SELECT r.id, r.code, r.name, r.type,
		       COUNT(p.id) AS total_ports,
		       COUNT(p.id) FILTER (WHERE p.status = 'USED') AS used_ports,
		       COALESCE(ROUND(COUNT(p.id) FILTER (WHERE p.status = 'USED')::numeric * 100
		           / NULLIF(COUNT(p.id) FILTER (WHERE p.status IN ('USED','IDLE')), 0), 2), 0) AS usage_rate
		FROM resources r
		LEFT JOIN ports p ON p.resource_id = r.id
		WHERE ($1 = '' OR r.type = $1)
		GROUP BY r.id, r.code, r.name, r.type
		ORDER BY usage_rate `+dir+`, r.id`, dim)
	if err != nil {
		return nil, fmt.Errorf("resource: capacity aggregate: %w", err)
	}
	defer rows.Close()
	out := make([]CapacityStat, 0)
	for rows.Next() {
		var st CapacityStat
		if err := rows.Scan(&st.ResourceID, &st.Code, &st.Name, &st.Type, &st.TotalPorts, &st.UsedPorts, &st.UsageRate); err != nil {
			return nil, fmt.Errorf("resource: scan capacity: %w", err)
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// CapacityAlertScan 一轮阈值预警:越限且无 OPEN 容量告警则产生(状态变化才告警);
// 回落且有 OPEN 则关闭(恢复同样是状态变化);状态不变的重复扫描两者皆不动,幂等不重复。
func (s *PGStore) CapacityAlertScan(ctx context.Context, thresholdPct float64, sink CapacityAlarmSink) (CapacityScanResult, error) {
	stats, err := s.Capacity(ctx, "", OrderUsageDesc)
	if err != nil {
		return CapacityScanResult{}, fmt.Errorf("resource: capacity scan: %w", err)
	}
	out := CapacityScanResult{Scanned: len(stats)}
	for _, st := range stats {
		has, err := sink.HasOpenCapacityAlarm(ctx, st.ResourceID)
		if err != nil {
			return out, fmt.Errorf("resource: capacity scan has-open r%d: %w", st.ResourceID, err)
		}
		over := st.UsageRate >= thresholdPct
		switch {
		case over && !has:
			if err := sink.CreateCapacityAlarm(ctx, st.ResourceID, capacityAlarmContent(st, thresholdPct)); err != nil {
				return out, fmt.Errorf("resource: capacity scan create r%d: %w", st.ResourceID, err)
			}
			out.Created++
		case !over && has:
			n, err := sink.CloseCapacityAlarms(ctx, st.ResourceID)
			if err != nil {
				return out, fmt.Errorf("resource: capacity scan close r%d: %w", st.ResourceID, err)
			}
			out.Resolved += int(n)
		}
	}
	return out, nil
}

// capacityAlarmContent 告警正文:对象/使用率/占用占比,后台告警列表直接可读。
func capacityAlarmContent(st CapacityStat, thresholdPct float64) string {
	return fmt.Sprintf("容量预警: %s(%s) 端口使用率 %.2f%% >= %.2f%%,占用 %d/总 %d", st.Name, st.Code, st.UsageRate, thresholdPct, st.UsedPorts, st.TotalPorts)
}
