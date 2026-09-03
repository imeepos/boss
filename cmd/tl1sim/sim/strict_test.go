package sim

import (
	"bytes"
	"context"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/provision/tl1"
)

// dialSim 登录会话工厂(复用 newTestServer)。
func dialSim(t *testing.T, mod func(*Options)) *tl1.Session {
	t.Helper()
	srv, opt := newTestServer(t, mod)
	s, err := tl1.Dial(context.Background(), tl1.Config{
		Addr: srv.Addr(), User: opt.User, Pass: opt.Pass,
		CmdTimeout: 5 * time.Second, DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// simDo 断言无传输错误后返回响应。
func simDo(t *testing.T, s *tl1.Session, c tl1.Command) *tl1.Response {
	t.Helper()
	resp, err := s.Do(context.Background(), c)
	if err != nil {
		t.Fatalf("Do %s tag=%q: %v", c.Verb, c.Tag, err)
	}
	return resp
}

// wantDeny 断言 DENY 错误码且 ctag 原样回显(客户端匹配不漂移)。
func wantDeny(t *testing.T, resp *tl1.Response, ctag string, en int) {
	t.Helper()
	if resp.Completion != "DENY" || resp.EN != en {
		t.Fatalf("ctag=%s got completion=%s EN=%d ENDESC=%s; want DENY EN=%d", resp.CTag, resp.Completion, resp.EN, resp.ENDESC, en)
	}
	if resp.CTag != ctag {
		t.Fatalf("DENY ctag echo=%q want %q", resp.CTag, ctag)
	}
}

// wantCompld 断言 COMPLD EN=0 且 ctag 原样回显。
func wantCompld(t *testing.T, resp *tl1.Response, ctag string) {
	t.Helper()
	if resp.Completion != "COMPLD" || resp.EN != 0 {
		t.Fatalf("ctag=%s got completion=%s EN=%d ENDESC=%s", resp.CTag, resp.Completion, resp.EN, resp.ENDESC)
	}
	if resp.CTag != ctag {
		t.Fatalf("COMPLD ctag echo=%q want %q", resp.CTag, ctag)
	}
}

// captureSimLogs 收编全局 log(测试串行,不并行使用)。
func captureSimLogs(fn func()) string {
	var buf bytes.Buffer
	out := log.Writer()
	flags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(out)
		log.SetFlags(flags)
	}()
	fn()
	return buf.String()
}

// addONUCmd 合规 ADD-ONU 报文;mod 允许逐项破坏以构造负例。
func addONUCmd(onuid string, mod func(*tl1.Command)) tl1.Command {
	c := tl1.Command{
		Verb: "ADD-ONU", Tag: "ADDONT",
		Access: []tl1.KV{{K: "OLTID", V: "10.0.0.9"}, {K: "PONID", V: "NA-0-7-5"}},
		Payload: []tl1.KV{
			{K: "AUTHTYPE", V: "LOID"}, {K: "ONUID", V: onuid}, {K: "ONUNO", V: "9"},
			{K: "DESC", V: "PRV-ORD-S"}, {K: "ONUTYPE", V: "FTTH"},
		},
	}
	if mod != nil {
		mod(&c)
	}
	return c
}

// ponvlanCmd 合规 ADD-PONVLAN 报文(tag=Internet 带 SVLAN;mod 构造负例/TR069)。
func ponvlanCmd(onuid string, mod func(*tl1.Command)) tl1.Command {
	c := tl1.Command{
		Verb: "ADD-PONVLAN", Tag: "Internet",
		Access: []tl1.KV{
			{K: "OLTID", V: "10.0.0.9"}, {K: "PONID", V: "NA-0-7-5"},
			{K: "ONUIDTYPE", V: "LOID"}, {K: "ONUID", V: onuid},
		},
		Payload: []tl1.KV{
			{K: "SVLAN", V: "1000"}, {K: "CVLAN", V: "100"}, {K: "UV", V: "100"},
			{K: "DESC", V: "PRV-ORD-S-Internet"}, {K: "SCOS", V: "5"},
		},
	}
	if mod != nil {
		mod(&c)
	}
	return c
}

// dropKV 删除指定键(负例构造)。
func dropKV(kvs []tl1.KV, key string) []tl1.KV {
	out := make([]tl1.KV, 0, len(kvs))
	for _, kv := range kvs {
		if kv.K != key {
			out = append(out, kv)
		}
	}
	return out
}

// kvVal 取指定键现值(负例引用同值构造错放)。
func kvVal(kvs []tl1.KV, key string) string {
	for _, kv := range kvs {
		if kv.K == key {
			return kv.V
		}
	}
	return ""
}

// TestStrictTRCCTagRejected TRC<n> 占位 ctag 一律 DENY(写/查动词皆拒),留 STRICT DENY 观测日志。
func TestStrictTRCCTagRejected(t *testing.T) {
	s := dialSim(t, nil)
	var resp *tl1.Response
	logs := captureSimLogs(func() {
		resp = simDo(t, s, addONUCmd("loid-trc", func(c *tl1.Command) { c.Tag = "TRC7" }))
	})
	wantDeny(t, resp, "TRC7", enBadCtag)
	for _, want := range []string{"STRICT DENY", "TRC7", "ADD-ONU"} {
		if !strings.Contains(logs, want) {
			t.Fatalf("logs missing %q: %s", want, logs)
		}
	}
	// 查询动词同样拒绝:TRC 占位是全局规则,不限写命令。
	resp = simDo(t, s, tl1.Command{Verb: "LST-ONU", Tag: "TRC9", Access: []tl1.KV{{K: "OLTID", V: "10.0.0.9"}}})
	wantDeny(t, resp, "TRC9", enBadCtag)
}

// TestStrictADDONUContract ADD-ONU 契约矩阵:错误 ctag/缺键/ONUID 错段位 DENY,合规 COMPLD 回显。
func TestStrictADDONUContract(t *testing.T) {
	s := dialSim(t, nil)
	drop := func(key string) func(*tl1.Command) {
		return func(c *tl1.Command) {
			if key == "OLTID" || key == "PONID" {
				c.Access = dropKV(c.Access, key)
				return
			}
			c.Payload = dropKV(c.Payload, key)
		}
	}
	cases := []struct {
		name string
		mod  func(*tl1.Command)
		en   int
	}{
		{"svc ctag mismatch", func(c *tl1.Command) { c.Tag = "Internet" }, enBadCtag},
		{"access no OLTID", drop("OLTID"), enMissField},
		{"access no PONID", drop("PONID"), enMissField},
		{"payload no AUTHTYPE", drop("AUTHTYPE"), enMissField},
		{"payload no ONUID", drop("ONUID"), enMissField},
		{"payload no ONUNO", drop("ONUNO"), enMissField},
		{"payload no DESC", drop("DESC"), enMissField},
		{"payload no ONUTYPE", drop("ONUTYPE"), enMissField},
		{"ONUID in access", func(c *tl1.Command) {
			c.Access = append(c.Access, tl1.KV{K: "ONUID", V: kvVal(c.Payload, "ONUID")})
		}, enSegMisplace},
	}
	for i, tc := range cases {
		c := addONUCmd("loid-a"+strconv.Itoa(i), tc.mod)
		wantDeny(t, simDo(t, s, c), c.Tag, tc.en)
	}
	// 会话自增 ctag(B%06d)不是手册业务位,同样拒绝;回显自增 ctag。
	resp := simDo(t, s, addONUCmd("loid-ab", func(c *tl1.Command) { c.Tag = "" }))
	if resp.Completion != "DENY" || resp.EN != enBadCtag || !strings.HasPrefix(resp.CTag, "B") {
		t.Fatalf("session ctag: completion=%s EN=%d ctag=%s", resp.Completion, resp.EN, resp.CTag)
	}
	// 正例:ADDONT 全字段合规 → COMPLD,ctag 原样回显。
	wantCompld(t, simDo(t, s, addONUCmd("loid-ok", nil)), "ADDONT")
}

// TestStrictADDPONVLANContract ADD-PONVLAN 契约矩阵:ctag 限 Internet/TR069,
// access 需 ONUIDTYPE/ONUID,payload 至少 CVLAN/UV,ONUID 不得落 payload。
func TestStrictADDPONVLANContract(t *testing.T) {
	s := dialSim(t, nil)
	wantCompld(t, simDo(t, s, addONUCmd("loid-pv", nil)), "ADDONT")
	drop := func(key string) func(*tl1.Command) {
		return func(c *tl1.Command) {
			if key == "ONUIDTYPE" || key == "ONUID" || key == "OLTID" || key == "PONID" {
				c.Access = dropKV(c.Access, key)
				return
			}
			c.Payload = dropKV(c.Payload, key)
		}
	}
	cases := []struct {
		name string
		mod  func(*tl1.Command)
		en   int
	}{
		{"onu ctag mismatch", func(c *tl1.Command) { c.Tag = "ADDONT" }, enBadCtag},
		{"access no ONUIDTYPE", drop("ONUIDTYPE"), enMissField},
		{"access no ONUID", drop("ONUID"), enMissField},
		{"access no OLTID", drop("OLTID"), enMissField},
		{"payload no CVLAN", drop("CVLAN"), enMissField},
		{"payload no UV", drop("UV"), enMissField},
		{"ONUID in payload", func(c *tl1.Command) {
			c.Payload = append(c.Payload, tl1.KV{K: "ONUID", V: kvVal(c.Access, "ONUID")})
		}, enSegMisplace},
	}
	for _, tc := range cases {
		c := ponvlanCmd("loid-pv", tc.mod)
		wantDeny(t, simDo(t, s, c), c.Tag, tc.en)
	}
	// 会话自增 ctag 同样拒绝。
	resp := simDo(t, s, ponvlanCmd("loid-pv", func(c *tl1.Command) { c.Tag = "" }))
	if resp.Completion != "DENY" || resp.EN != enBadCtag || !strings.HasPrefix(resp.CTag, "B") {
		t.Fatalf("session ctag: completion=%s EN=%d ctag=%s", resp.Completion, resp.EN, resp.CTag)
	}
	// 正例:Internet(双层)与 TR069(单层,无 SVLAN)均 COMPLD 且 ctag 回显。
	wantCompld(t, simDo(t, s, ponvlanCmd("loid-pv", nil)), "Internet")
	tr069 := ponvlanCmd("loid-pv", func(c *tl1.Command) {
		c.Tag = "TR069"
		c.Payload = []tl1.KV{
			{K: "CVLAN", V: "100"}, {K: "UV", V: "100"}, {K: "DESC", V: "PRV-ORD-S-TR069"},
		}
	})
	wantCompld(t, simDo(t, s, tr069), "TR069")
}

// TestStrictDisabledCompat StrictTags=false 兼容调试:B 自增/TRC 占位 ctag 放行,无 STRICT DENY 日志。
func TestStrictDisabledCompat(t *testing.T) {
	loose := false
	s := dialSim(t, func(o *Options) { o.StrictTags = &loose })
	var r1, r2 *tl1.Response
	logs := captureSimLogs(func() {
		r1 = simDo(t, s, addONUCmd("loid-loose1", func(c *tl1.Command) { c.Tag = "" })) // B 自增 ctag
		r2 = simDo(t, s, addONUCmd("loid-loose2", func(c *tl1.Command) { c.Tag = "TRC1" }))
	})
	if r1.Completion != "COMPLD" || !strings.HasPrefix(r1.CTag, "B") {
		t.Fatalf("loose B ctag: %+v", r1)
	}
	wantCompld(t, r2, "TRC1")
	// 关严格后宽容 ADD-PONVLAN(B ctag、ONUID 仅在 access)照常建流。
	r3 := simDo(t, s, ponvlanCmd("loid-loose1", func(c *tl1.Command) { c.Tag = "" }))
	if r3.Completion != "COMPLD" || !strings.HasPrefix(r3.CTag, "B") {
		t.Fatalf("loose ponvlan: %+v", r3)
	}
	if strings.Contains(logs, "STRICT DENY") {
		t.Fatalf("strict disabled but STRICT DENY logged: %s", logs)
	}
}
