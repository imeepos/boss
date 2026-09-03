package sim

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// onuRec 一条 ONU 注册记录,键为 oltid|ponid|onuid(ADD-ONU 重复判定同键)。
type onuRec struct {
	oltid, ponid, onuid   string
	onuno                 int
	authtype, onutype     string
	desc                  string
	adminState, operState string
	cfgstat               string
}

// svcRec 一条 Service Port 记录,键为 oltid|ponid|onuid|desc。
type svcRec struct {
	oltid, ponid, onuid, desc string
	svlan, cvlan, uv          int
	scos, ccos                int
	hasSVLAN                  bool
}

// state 仿真网元内存态:登录位、ONU/业务流表、故障注入参数。
type state struct {
	mu       sync.Mutex
	loggedIn bool
	onus     map[[3]string]*onuRec
	svcs     map[[4]string]*svcRec
	denyNext string
	strict   bool
	opt      Options
	rec      *record
}

func newState(opt Options, rec *record) *state {
	strict := true // 手册严格校验默认开启;显式 StrictTags=false 仅兼容调试
	if opt.StrictTags != nil {
		strict = *opt.StrictTags
	}
	return &state{
		onus:     map[[3]string]*onuRec{},
		svcs:     map[[4]string]*svcRec{},
		denyNext: opt.DenyNext,
		strict:   strict,
		opt:      opt,
		rec:      rec,
	}
}

// cmdReq 解析后的入站指令(Build 的逆)。
type cmdReq struct {
	verb, ctag      string
	access, payload map[string]string
}

// parseCmd 按 "VERB::access:ctag::payload;" 定长切分;Build 恒产出该形状。
func parseCmd(line string) (cmdReq, error) {
	s := strings.TrimSuffix(strings.TrimSpace(line), ";")
	parts := strings.Split(s, ":")
	if len(parts) != 6 || parts[0] == "" {
		return cmdReq{}, fmt.Errorf("malformed command %q", line)
	}
	return cmdReq{
		verb:    parts[0],
		ctag:    parts[3],
		access:  parseKVs(parts[2]),
		payload: parseKVs(parts[5]),
	}, nil
}

// parseKVs 切 "K=V,K=V";TL1 白名单字符集不含 :/=,,首见分隔即安全。
func parseKVs(s string) map[string]string {
	m := map[string]string{}
	if s == "" {
		return m
	}
	for _, kv := range strings.Split(s, ",") {
		i := strings.Index(kv, "=")
		if i <= 0 {
			continue
		}
		m[kv[:i]] = kv[i+1:]
	}
	return m
}

// handle 处理一条指令:留痕打日志,按需注入 DELAY,返回按序响应帧与是否收尾断连。
func (st *state) handle(line string) ([]string, bool) {
	cmd, err := parseCmd(line)
	if err != nil {
		log.Printf("[tl1sim] RECV malformed %q: %v", line, err)
		return []string{denyFrame("0", enBadCmd, "报文格式非法。")}, false
	}
	log.Printf("[tl1sim] RECV %s", line)
	st.rec.add(cmd.verb, line)
	st.mu.Lock()
	defer st.mu.Unlock()
	var frames []string
	// LOGIN 豁免:Session.login 只读一次响应,不循环等同 ctag 最终帧(PDF §11.1)。
	if st.opt.DelayMs > 0 && cmd.verb != "LOGIN" {
		frames = append(frames, delayFrame(cmd.ctag))
		time.Sleep(time.Duration(st.opt.DelayMs) * time.Millisecond)
	}
	final, closeConn := st.dispatch(cmd)
	return append(frames, final), closeConn
}

// dispatch 按动词分派;严格手册校验最先(协议先于会话/业务,违规报文不消耗
// denyNext 注入额度),注入次之,最后业务逻辑。
func (st *state) dispatch(cmd cmdReq) (string, bool) {
	if deny := st.strictCheck(cmd); deny != "" {
		return deny, false
	}
	if st.denyNext == cmd.verb {
		st.denyNext = ""
		return denyFrame(cmd.ctag, enDenyInject, "仿真故障注入 DENY。"), false
	}
	switch cmd.verb {
	case "LOGIN":
		return st.login(cmd), false
	case "SHAKEHAND", "LOGOUT":
		return st.bye(cmd)
	case "ADD-ONU":
		return st.guarded(cmd, st.addONU)
	case "ADD-PONVLAN":
		return st.guarded(cmd, st.addPONVLAN)
	case "DEL-ONU":
		return st.guarded(cmd, st.delONU)
	case "CFG-ONU":
		return st.guarded(cmd, st.cfgONU)
	case "LST-ONU":
		return st.guarded(cmd, st.lstONU)
	case "LST-PONVLAN":
		return st.guarded(cmd, st.lstPONVLAN)
	case "LST-ONUSTATE":
		return st.guarded(cmd, st.lstONUState)
	default:
		return denyFrame(cmd.ctag, enBadCmd, "不支持的命令。"), false
	}
}

// guarded 未登录一律 DENY(除 LOGIN/SHAKEHAND/LOGOUT),登录后走业务 handler。
func (st *state) guarded(cmd cmdReq, fn func(cmdReq) string) (string, bool) {
	if !st.loggedIn {
		return denyFrame(cmd.ctag, enNotLoggedIn, "未登录或会话失效。"), false
	}
	return fn(cmd), false
}

func (st *state) bye(cmd cmdReq) (string, bool) {
	if !st.loggedIn {
		return denyFrame(cmd.ctag, enNotLoggedIn, "未登录或会话失效。"), false
	}
	if cmd.verb == "LOGOUT" {
		st.loggedIn = false
		return compldFrame(cmd.ctag), true
	}
	return compldFrame(cmd.ctag), false
}

// login 校验 UN/PWD(错→DENY EN=76546031,PDF 登录失败示例)。
func (st *state) login(cmd cmdReq) string {
	if cmd.payload["UN"] == st.opt.User && cmd.payload["PWD"] == st.opt.Pass {
		st.loggedIn = true
		return compldFrame(cmd.ctag)
	}
	return denyFrame(cmd.ctag, enAuthFail, "EMS operation failed ErrorCode: 76546031 (用户名或密码错误。)")
}

// addONU 建卡;oltid|ponid|onuid 同键重复→DENY。
func (st *state) addONU(cmd cmdReq) string {
	oltid, ponid, onuid := cmd.access["OLTID"], cmd.access["PONID"], cmd.payload["ONUID"]
	key := [3]string{oltid, ponid, onuid}
	if _, ok := st.onus[key]; ok {
		return denyFrame(cmd.ctag, enONUExists, "ONU 已存在。")
	}
	onuno, _ := strconv.Atoi(cmd.payload["ONUNO"])
	authtype := cmd.payload["AUTHTYPE"]
	if authtype == "" {
		authtype = "LOID"
	}
	st.onus[key] = &onuRec{
		oltid: oltid, ponid: ponid, onuid: onuid, onuno: onuno,
		authtype: authtype, onutype: cmd.payload["ONUTYPE"], desc: cmd.payload["DESC"],
		adminState: "UP", operState: st.opt.ONUOperState, cfgstat: "NORMAL",
	}
	return compldFrame(cmd.ctag)
}

// addPONVLAN 建业务流;要求 ONU 已注册,同键重复→DENY。
func (st *state) addPONVLAN(cmd cmdReq) string {
	oltid, ponid, onuid := cmd.access["OLTID"], cmd.access["PONID"], cmd.access["ONUID"]
	if st.onus[[3]string{oltid, ponid, onuid}] == nil {
		return denyFrame(cmd.ctag, enONUMissing, "ONU 未注册,禁止配置业务流。")
	}
	desc := cmd.payload["DESC"]
	key := [4]string{oltid, ponid, onuid, desc}
	if _, ok := st.svcs[key]; ok {
		return denyFrame(cmd.ctag, enSvcExists, "Service Port 已存在。")
	}
	st.svcs[key] = &svcRec{
		oltid: oltid, ponid: ponid, onuid: onuid, desc: desc,
		svlan:    atoiOr(cmd.payload["SVLAN"]),
		cvlan:    atoiOr(cmd.payload["CVLAN"]),
		uv:       atoiOr(cmd.payload["UV"]),
		scos:     atoiOr(cmd.payload["SCOS"]),
		ccos:     atoiOr(cmd.payload["CCOS"]),
		hasSVLAN: cmd.payload["SVLAN"] != "",
	}
	return compldFrame(cmd.ctag)
}

// delONU 删卡并级联删除其业务流。
func (st *state) delONU(cmd cmdReq) string {
	onuid := firstNonEmpty(cmd.payload["ONUID"], cmd.access["ONUID"])
	key := [3]string{cmd.access["OLTID"], cmd.access["PONID"], onuid}
	if _, ok := st.onus[key]; !ok {
		return denyFrame(cmd.ctag, enONUMissing, "ONU 不存在。")
	}
	delete(st.onus, key)
	for k, s := range st.svcs {
		if [3]string{s.oltid, s.ponid, s.onuid} == key {
			delete(st.svcs, k)
		}
	}
	return compldFrame(cmd.ctag)
}

// cfgONU 改卡:仿真语义为配置下发生效,CFGSTAT 置 CONFIG。
func (st *state) cfgONU(cmd cmdReq) string {
	rec := st.findONU(cmd)
	if rec == nil {
		return denyFrame(cmd.ctag, enONUMissing, "ONU 不存在。")
	}
	rec.cfgstat = "CONFIG"
	if v := cmd.payload["ONUTYPE"]; v != "" {
		rec.onutype = v
	}
	return compldFrame(cmd.ctag)
}

// findONU ONUID 兼容 access/payload 双位(各命令参数位随 PDF 相异,仿真侧宽容解析)。
func (st *state) findONU(cmd cmdReq) *onuRec {
	onuid := firstNonEmpty(cmd.payload["ONUID"], cmd.access["ONUID"])
	return st.onus[[3]string{cmd.access["OLTID"], cmd.access["PONID"], onuid}]
}

func atoiOr(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
