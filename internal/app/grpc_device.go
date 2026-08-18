package app

// gRPC device/v1 服务端:指标批量上报 + Trap 告警实时推送(collector→server)。
// 复用 device.Collector 阈值告警链路,资源编码 → 资源ID 经 ListResources 解析。

import (
	"context"
	"strconv"
	"time"

	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"
	devicev1 "github.com/ymm-001/boss/api/proto/boss/device/v1"

	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/resource"
)

// deviceGRPC DeviceIngestService 实现。
type deviceGRPC struct {
	devicev1.UnimplementedDeviceIngestServiceServer
	dev                device.DeviceService
	alarm              device.AlarmService
	res                resource.ResourceService
	packetLossAlarmPct float64
}

// ReportMetrics SNMP 周期指标批量上报,返回受理条数。
func (s *deviceGRPC) ReportMetrics(ctx context.Context, req *devicev1.ReportMetricsRequest) (*devicev1.OpResponse, error) {
	ids := s.resourceIDs(ctx)
	collector := &device.Collector{
		Dev: s.dev, Alarm: s.alarm, PacketLossAlarmPct: &s.packetLossAlarmPct,
	}
	accepted := 0
	for _, smp := range req.Samples {
		rid, ok := ids[smp.ResourceCode]
		if !ok {
			continue // 未知资源跳过(不污染受理计数)
		}
		sample := sampleFromGauges(smp)
		sample.ResourceID = rid
		sample.CollectedAt = time.Unix(smp.CollectedAt, 0)
		if err := collector.Ingest(ctx, sample); err != nil {
			continue
		}
		accepted++
	}
	return &devicev1.OpResponse{Code: commonv1.Code_CODE_OK, Accepted: int32(accepted)}, nil
}

// ReportTrap Trap 告警实时上报,触发告警规则入库。
func (s *deviceGRPC) ReportTrap(ctx context.Context, req *devicev1.Trap) (*devicev1.OpResponse, error) {
	ids := s.resourceIDs(ctx)
	rid, ok := ids[req.ResourceCode]
	if !ok {
		return &devicev1.OpResponse{Code: commonv1.Code_CODE_OK, Accepted: 0}, nil
	}
	content := req.Oid
	if len(req.Vars) > 0 {
		content = req.Oid + " " + varsSummary(req.Vars)
	}
	no := "ALM-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := s.alarm.CreateAlarm(ctx, device.Alarm{
		AlarmNo: no, ResourceID: rid, Level: levelOf(req.Severity), Source: "device",
		Content: content, Status: "OPEN", CreatedAt: time.Now(),
	}); err != nil {
		return &devicev1.OpResponse{Code: commonv1.Code_CODE_INTERNAL, Accepted: 0}, nil
	}
	return &devicev1.OpResponse{Code: commonv1.Code_CODE_OK, Accepted: 1}, nil
}

// resourceIDs 资源编码 → 资源ID 映射(每请求一次)。
func (s *deviceGRPC) resourceIDs(ctx context.Context) map[string]int64 {
	out := make(map[string]int64)
	list, err := s.res.ListResources(ctx)
	if err != nil {
		return out
	}
	for _, r := range list {
		out[r.Code] = r.ID
	}
	return out
}

// sampleFromGauges gauges → Sample(光功率/丢包率,键名容忍中英双写)。
func sampleFromGauges(smp *devicev1.MetricSample) device.Sample {
	var optical, loss *float64
	for k, v := range smp.Gauges {
		switch k {
		case "optical_power", "光功率", "opticalPower":
			val := v
			optical = &val
		case "packet_loss", "丢包率", "packetLoss":
			val := v
			loss = &val
		}
	}
	status := "ONLINE"
	if loss == nil {
		status = "OFFLINE"
	}
	return device.Sample{OpticalPower: optical, PacketLoss: loss, Status: status}
}

// levelOf 告警 severity → 告警级别(契约 INFO/WARNING/CRITICAL)。
func levelOf(severity string) string {
	switch severity {
	case "WARNING":
		return "WARNING"
	case "CRITICAL":
		return "CRITICAL"
	default:
		return "INFO"
	}
}

// varsSummary Trap 变量绑定摘要(键=值 拼接,定位可读)。
func varsSummary(vars map[string]string) string {
	out := ""
	for k, v := range vars {
		out += k + "=" + v + " "
	}
	return out
}
