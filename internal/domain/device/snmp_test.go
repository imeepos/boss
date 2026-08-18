package device

// SNMPPoller 单测:注入桩 getter 验证 OID→Sample 映射(光功率/丢包率/状态)与单目标容错。

import (
	"context"
	"testing"
)

type fakeGetter struct {
	vals map[string]float64
	err  error
	oids []string
}

func (f *fakeGetter) Get(oids []string) (map[string]float64, error) {
	f.oids = oids
	return f.vals, f.err
}

func TestSNMPPoller_Poll(t *testing.T) {
	opticalOID := "1.3.6.1.4.1.100.1"
	lossOID := "1.3.6.1.4.1.100.2"
	statusOID := "1.3.6.1.2.1.2.2.1.8"

	t.Run("采集映射与状态", func(t *testing.T) {
		g := &fakeGetter{vals: map[string]float64{
			opticalOID: -24.5, lossOID: 12.0, statusOID: 1,
		}}
		p := &SNMPPoller{
			Targets:    []Target{{ResourceID: 1, Code: "OLT-01", Host: "10.0.0.1"}},
			OpticalOID: opticalOID, PacketLossOID: lossOID, StatusOID: statusOID,
			getter: func(string) snmpGetter { return g },
		}
		samples, err := p.Poll(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(samples) != 1 {
			t.Fatalf("samples=%d", len(samples))
		}
		s := samples[0]
		if s.ResourceID != 1 || s.Status != "ONLINE" || s.OpticalPower == nil || *s.OpticalPower != -24.5 || s.PacketLoss == nil || *s.PacketLoss != 12 {
			t.Fatalf("sample=%+v", s)
		}
	})

	t.Run("down 状态映射 OFFLINE", func(t *testing.T) {
		g := &fakeGetter{vals: map[string]float64{statusOID: 2}}
		p := &SNMPPoller{
			Targets:    []Target{{ResourceID: 2, Code: "OLT-02", Host: "10.0.0.2"}},
			OpticalOID: opticalOID, PacketLossOID: lossOID, StatusOID: statusOID,
			getter: func(string) snmpGetter { return g },
		}
		samples, _ := p.Poll(context.Background())
		if samples[0].Status != "OFFLINE" || samples[0].OpticalPower != nil {
			t.Fatalf("sample=%+v", samples[0])
		}
	})

	t.Run("单目标失败不阻断整轮", func(t *testing.T) {
		fail := &fakeGetter{err: context.DeadlineExceeded}
		ok := &fakeGetter{vals: map[string]float64{statusOID: 1}}
		p := &SNMPPoller{
			Targets: []Target{
				{ResourceID: 1, Code: "OLT-01", Host: "10.0.0.1"},
				{ResourceID: 2, Code: "OLT-02", Host: "10.0.0.2"},
			},
			OpticalOID: opticalOID, PacketLossOID: lossOID, StatusOID: statusOID,
			getter: func(host string) snmpGetter {
				if host == "10.0.0.1" {
					return fail
				}
				return ok
			},
		}
		samples, err := p.Poll(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(samples) != 1 || samples[0].ResourceID != 2 {
			t.Fatalf("samples=%+v", samples)
		}
	})
}

func TestSplitHostPort(t *testing.T) {
	cases := []struct {
		in   string
		host string
		port uint16
	}{
		{"10.0.0.1:161", "10.0.0.1", 161},
		{"10.0.0.1", "10.0.0.1", 161},
		{"[::1]:1661", "[::1]", 1661},
	}
	for _, c := range cases {
		h, p := splitHostPort(c.in)
		if h != c.host || p != c.port {
			t.Fatalf("splitHostPort(%q)=(%q,%d), want (%q,%d)", c.in, h, p, c.host, c.port)
		}
	}
}

func TestSnmpToFloat(t *testing.T) {
	if snmpToFloat(int(5)) != 5 || snmpToFloat(uint64(7)) != 7 || snmpToFloat([]byte(" 3 ")) != 3 {
		t.Fatal("snmpToFloat mismatch")
	}
	if snmpToFloat([]byte("abc")) != 0 {
		t.Fatal("non-numeric should be 0")
	}
}
