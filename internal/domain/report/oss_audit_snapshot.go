package report

// 资源台账稽核每日快照(P5-W3):稽核 SQL 归 resource 域(InventoryAuditor),
// 本层只管把结果落 report_snapshots(period=oss-audit-daily,同日幂等覆盖),
// 窗口口径与 DailyRecon 一致(当日 00:00 起 24h)。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// PeriodOSSAuditDaily 资源稽核快照周期键(与 daily/recon-daily 区分)。
const PeriodOSSAuditDaily = "oss-audit-daily"

// SaveOSSAudit 稽核结果落当日快照(同日覆盖);payload 任意 JSON 可序列化结构
// (通常 resource.AuditReport,report 不反向依赖 resource,经 JSON 解耦)。
func (r *ReportService) SaveOSSAudit(ctx context.Context, payload any, at time.Time) (*Snapshot, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("report: oss-audit marshal: %w", err)
	}
	start := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
	snap := &Snapshot{
		Period: PeriodOSSAuditDaily, WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
		Payload: raw, CreatedAt: at,
	}
	if err := r.St.UpsertSnapshot(ctx, snap); err != nil {
		return nil, fmt.Errorf("report: oss-audit save: %w", err)
	}
	return snap, nil
}
