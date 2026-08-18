package device

// SNMPPoller 真实 SNMP 采集源(债务偿还:替换 cmd/collector 的 noopPoller 桩)。
// 生产走 gosnmp v2c GET;OID 可配(厂商 profile 不同),采集值映射为 Sample 交 Collector 入库/告警。

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// Target 采集目标(OLT/ONU,资源编码定位资源ID,Host 承载网元地址)。
type Target struct {
	ResourceID int64
	Code       string
	Host       string // host 或 host:port
}

// snmpGetter SNMP GET 抽象(生产走 gosnmp UDP;单测注入桩保证确定性)。
type snmpGetter interface {
	Get(oids []string) (map[string]float64, error)
}

// SNMPPoller 按目标批量 GET 三项指标 OID,产出 Samples。
type SNMPPoller struct {
	Targets       []Target
	Community     string
	OpticalOID    string
	PacketLossOID string
	StatusOID     string
	// getter 缺省用 gosnmpGetter;单测覆盖。
	getter func(host string) snmpGetter
}

// Poll 逐目标采集;单目标失败不阻断整轮(该目标本轮无样本)。
func (p *SNMPPoller) Poll(ctx context.Context) ([]Sample, error) {
	out := make([]Sample, 0, len(p.Targets))
	now := time.Now()
	for _, t := range p.Targets {
		g := p.getterFor(t.Host)
		vals, err := g.Get([]string{p.OpticalOID, p.PacketLossOID, p.StatusOID})
		if err != nil {
			continue
		}
		out = append(out, p.sampleOf(t, vals, now))
	}
	return out, nil
}

// getterFor 构造目标采集器;未注入 getter 时用真实 gosnmp。
func (p *SNMPPoller) getterFor(host string) snmpGetter {
	if p.getter != nil {
		return p.getter(host)
	}
	return gosnmpGetter{community: p.Community, host: host, timeout: 2 * time.Second}
}

// sampleOf OID 值 → Sample(状态:1 up/2 down/其它 fault)。
func (p *SNMPPoller) sampleOf(t Target, vals map[string]float64, at time.Time) Sample {
	var optical, loss *float64
	if v, ok := vals[p.OpticalOID]; ok {
		val := v
		optical = &val
	}
	if v, ok := vals[p.PacketLossOID]; ok {
		val := v
		loss = &val
	}
	status := "FAULT"
	if v, ok := vals[p.StatusOID]; ok {
		status = statusFromOper(v)
	}
	return Sample{ResourceID: t.ResourceID, OpticalPower: optical, PacketLoss: loss, Status: status, CollectedAt: at}
}

// statusFromOper ifOperStatus → resource.status 1→ONLINE 2→OFFLINE。
func statusFromOper(v float64) string {
	switch int(v) {
	case 1:
		return "ONLINE"
	case 2:
		return "OFFLINE"
	default:
		return "FAULT"
	}
}

// gosnmpGetter 真实 SNMP v2c GET 客户端(每目标短连接)。
type gosnmpGetter struct {
	community string
	host      string
	timeout   time.Duration
}

// Get 对给定 OID 列表做一次 v2c GET,返回 oid→数值(统一转 float64)。
func (g gosnmpGetter) Get(oids []string) (map[string]float64, error) {
	host, port := splitHostPort(g.host)
	client := &gosnmp.GoSNMP{
		Target: host, Port: port, Community: g.community,
		Version: gosnmp.Version2c, Timeout: g.timeout, Retries: 1,
	}
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("device: snmp connect: %w", err)
	}
	defer client.Conn.Close()
	res, err := client.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("device: snmp get: %w", err)
	}
	out := make(map[string]float64, len(res.Variables))
	for _, v := range res.Variables {
		out[v.Name] = snmpToFloat(v.Value)
	}
	return out, nil
}

// splitHostPort host 或 host:port → (host, port),缺省 161。
func splitHostPort(addr string) (string, uint16) {
	if i := strings.LastIndex(addr, ":"); i > 0 && !strings.Contains(addr[i+1:], ":") {
		if p, err := strconv.Atoi(addr[i+1:]); err == nil {
			return addr[:i], uint16(p)
		}
	}
	return addr, 161
}

// snmpToFloat 常见 SNMP 数值类型 → float64(不可识别返回 0)。
func snmpToFloat(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	case []byte:
		if f, err := strconv.ParseFloat(strings.TrimSpace(string(n)), 64); err == nil {
			return f
		}
	}
	return 0
}
