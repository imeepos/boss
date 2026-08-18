package app

// gRPC device/v1 契约单测:指标批量上报(阈值告警)+ Trap 实时告警。

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"
	devicev1 "github.com/ymm-001/boss/api/proto/boss/device/v1"

	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/resource"
)

type stubDevice struct {
	device.DeviceService
	metrics []device.DeviceMetric
	err     error
}

func (s *stubDevice) AppendMetric(_ context.Context, m device.DeviceMetric) (int64, error) {
	s.metrics = append(s.metrics, m)
	return int64(len(s.metrics)), s.err
}

type stubAlarm struct {
	device.AlarmService
	alarms []device.Alarm
}

func (s *stubAlarm) CreateAlarm(_ context.Context, a device.Alarm) (int64, error) {
	s.alarms = append(s.alarms, a)
	return int64(len(s.alarms)), nil
}

type stubResourceGRPC struct {
	resource.ResourceService
	list []resource.Resource
}

func (s *stubResourceGRPC) ListResources(context.Context) ([]resource.Resource, error) {
	return s.list, nil
}

func deviceGRPCWith(dev *stubDevice, al *stubAlarm, res *stubResourceGRPC) *deviceGRPC {
	return &deviceGRPC{dev: dev, alarm: al, res: res, packetLossAlarmPct: defaultPacketLossAlarmPct}
}

func TestDeviceGRPC_ReportMetrics(t *testing.T) {
	ctx := context.Background()
	res := &stubResourceGRPC{list: []resource.Resource{{ID: 1, Code: "OLT-01"}, {ID: 2, Code: "OLT-02"}}}

	t.Run("越限样本入库并告警", func(t *testing.T) {
		dev := &stubDevice{}
		al := &stubAlarm{}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			devicev1.RegisterDeviceIngestServiceServer(s, deviceGRPCWith(dev, al, res))
		})
		resp, err := devicev1.NewDeviceIngestServiceClient(conn).ReportMetrics(ctx, &devicev1.ReportMetricsRequest{
			Samples: []*devicev1.MetricSample{{
				ResourceCode: "OLT-01",
				Gauges:       map[string]float64{"packet_loss": 12, "optical_power": -24},
				CollectedAt:  1750000000,
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Code != commonv1.Code_CODE_OK || resp.Accepted != 1 {
			t.Fatalf("resp=%+v", resp)
		}
		if len(dev.metrics) != 1 || dev.metrics[0].ResourceID != 1 || dev.metrics[0].PacketLoss == nil || *dev.metrics[0].PacketLoss != 12 {
			t.Fatalf("metrics=%+v", dev.metrics)
		}
		if len(al.alarms) != 1 || al.alarms[0].Level != "CRITICAL" {
			t.Fatalf("alarms=%+v", al.alarms)
		}
	})

	t.Run("未越限不告警", func(t *testing.T) {
		dev := &stubDevice{}
		al := &stubAlarm{}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			devicev1.RegisterDeviceIngestServiceServer(s, deviceGRPCWith(dev, al, res))
		})
		resp, err := devicev1.NewDeviceIngestServiceClient(conn).ReportMetrics(ctx, &devicev1.ReportMetricsRequest{
			Samples: []*devicev1.MetricSample{{
				ResourceCode: "OLT-01",
				Gauges:       map[string]float64{"packet_loss": 1, "optical_power": -22},
				CollectedAt:  1750000000,
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Accepted != 1 || len(al.alarms) != 0 {
			t.Fatalf("resp=%+v alarms=%d", resp, len(al.alarms))
		}
	})

	t.Run("未知资源跳过", func(t *testing.T) {
		dev := &stubDevice{}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			devicev1.RegisterDeviceIngestServiceServer(s, deviceGRPCWith(dev, &stubAlarm{}, res))
		})
		resp, err := devicev1.NewDeviceIngestServiceClient(conn).ReportMetrics(ctx, &devicev1.ReportMetricsRequest{
			Samples: []*devicev1.MetricSample{{ResourceCode: "OLT-NOPE", Gauges: map[string]float64{"packet_loss": 99}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Accepted != 0 || len(dev.metrics) != 0 {
			t.Fatalf("resp=%+v metrics=%d", resp, len(dev.metrics))
		}
	})
}

func TestDeviceGRPC_ReportTrap(t *testing.T) {
	ctx := context.Background()
	res := &stubResourceGRPC{list: []resource.Resource{{ID: 1, Code: "OLT-01"}}}

	t.Run("CRITICAL Trap 入库", func(t *testing.T) {
		al := &stubAlarm{}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			devicev1.RegisterDeviceIngestServiceServer(s, deviceGRPCWith(&stubDevice{}, al, res))
		})
		resp, err := devicev1.NewDeviceIngestServiceClient(conn).ReportTrap(ctx, &devicev1.Trap{
			ResourceCode: "OLT-01", Oid: "1.3.6.1.4.1.0.1", Severity: "CRITICAL",
			Vars: map[string]string{"ifIndex": "1", "ifStatus": "down"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Accepted != 1 {
			t.Fatalf("resp=%+v", resp)
		}
		if len(al.alarms) != 1 || al.alarms[0].Level != "CRITICAL" || al.alarms[0].ResourceID != 1 {
			t.Fatalf("alarms=%+v", al.alarms)
		}
	})

	t.Run("未知资源忽略", func(t *testing.T) {
		al := &stubAlarm{}
		conn := newBufConnServer(t, func(s *grpc.Server) {
			devicev1.RegisterDeviceIngestServiceServer(s, deviceGRPCWith(&stubDevice{}, al, res))
		})
		resp, err := devicev1.NewDeviceIngestServiceClient(conn).ReportTrap(ctx, &devicev1.Trap{ResourceCode: "OLT-NOPE", Severity: "INFO"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Accepted != 0 || len(al.alarms) != 0 {
			t.Fatalf("resp=%+v", resp)
		}
	})
}
