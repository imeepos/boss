package sim

import (
	"log"
	"regexp"
	"slices"
	"strings"
)

// 手册严格校验(默认开启,Options.StrictTags=false 仅兼容调试关闭):
// 客户端业务 ctag 契约(C1)在网元侧对账——TRC 留痕占位不得入 ctag 位,
// 写命令业务 ctag 与 access/payload 段位逐键核对,违规 DENY 并留观测日志。

// trcRe 留痕占位 ctag(TRC+数字):仅存在于客户端 trace,占据网元 ctag 位即非法。
var trcRe = regexp.MustCompile("^TRC[0-9]+$")

// strictRule 单个写命令的手册段位规则。
type strictRule struct {
	ctags       []string // 合法业务 ctag 集合
	accessKeys  []string // access 段必填键
	payloadKeys []string // payload 段必填键(至少)
	onuidNotIn  string   // ONUID 禁放段位: "access"|"payload"|""(不限)
}

// strictRules 手册报文契约表(与客户端 client.go 业务位同源):
// ADD-ONU ctag=ADDONT、ONUID 归 payload;ADD-PONVLAN ctag=服务名(Internet/TR069)、
// ONUID 归 access,payload 至少 CVLAN/UV。
var strictRules = map[string]strictRule{
	"ADD-ONU": {
		ctags:       []string{"ADDONT"},
		accessKeys:  []string{"OLTID", "PONID"},
		payloadKeys: []string{"AUTHTYPE", "ONUID", "ONUNO", "DESC", "ONUTYPE"},
		onuidNotIn:  "access",
	},
	"ADD-PONVLAN": {
		ctags:       []string{"Internet", "TR069"},
		accessKeys:  []string{"OLTID", "PONID", "ONUIDTYPE", "ONUID"},
		payloadKeys: []string{"CVLAN", "UV"},
		onuidNotIn:  "payload",
	},
}

// strictCheck 手册一致性校验:违规返回 DENY 帧,合规返回空串。
// 协议级检查先于登录态/故障注入/业务逻辑;DENY 帧回显原 ctag,客户端响应匹配不漂移。
func (st *state) strictCheck(cmd cmdReq) string {
	if !st.strict {
		return ""
	}
	if trcRe.MatchString(cmd.ctag) {
		return strictDeny(cmd, enBadCtag, "ctag 禁止 TRC 留痕占位。")
	}
	r, ok := strictRules[cmd.verb]
	if !ok {
		return ""
	}
	if !slices.Contains(r.ctags, cmd.ctag) {
		return strictDeny(cmd, enBadCtag, cmd.verb+" ctag 须为 "+strings.Join(r.ctags, " 或 ")+"。")
	}
	if miss := firstMissing(cmd.access, r.accessKeys, "access"); miss != "" {
		return strictDeny(cmd, enMissField, miss)
	}
	if miss := firstMissing(cmd.payload, r.payloadKeys, "payload"); miss != "" {
		return strictDeny(cmd, enMissField, miss)
	}
	if onuidMisplaced(cmd, r.onuidNotIn) {
		return strictDeny(cmd, enSegMisplace, "ONUID 段位错放:不得置于 "+r.onuidNotIn+" 段。")
	}
	return ""
}

// strictDeny 拒绝帧 + 可 grep 观测日志。
func strictDeny(cmd cmdReq, en int, desc string) string {
	log.Printf("[tl1sim] STRICT DENY verb=%s ctag=%s EN=%d %s", cmd.verb, cmd.ctag, en, desc)
	return denyFrame(cmd.ctag, en, desc)
}

// firstMissing 返回首个缺失/空值必填键的描述;全齐返回空串。
func firstMissing(m map[string]string, keys []string, seg string) string {
	for _, k := range keys {
		if m[k] == "" {
			return seg + " 段缺少必填字段 " + k + "。"
		}
	}
	return ""
}

// onuidMisplaced ONUID 出现在禁放段位(非空值即违例)。
func onuidMisplaced(cmd cmdReq, seg string) bool {
	if seg == "" {
		return false
	}
	m := cmd.payload
	if seg == "access" {
		m = cmd.access
	}
	return m["ONUID"] != ""
}
