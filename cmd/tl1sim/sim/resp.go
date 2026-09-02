package sim

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 响应帧构造与查询出表,格式逐字节对齐 PDF §10.3/§13.1.2/§13.2.1/§15.4.4 示例:
// 头行缩进 3 空格,M 行双空格对齐,EN/ENDESC 行缩进 3 空格,终止符 ; 独行。

const sid = "HW_127.0.0.1"

// sim 自定义错误码段;仅 76546031 对齐 PDF 登录失败示例,其余为仿真内部分类。
const (
	enAuthFail    = 76546031
	enNotLoggedIn = 76546100
	enONUExists   = 76546101
	enONUMissing  = 76546102
	enDenyInject  = 76546103
	enBadCmd      = 76546104
	enSvcExists   = 76546105
)

// nowStr 响应头时间戳,格式对齐 PDF 示例。
func nowStr() string { return time.Now().Format("2006-01-02 15:04:05") }

// opFrame 操作类响应帧。
func opFrame(ctag, compl string, en int, desc string) string {
	return fmt.Sprintf("   %s %s\nM  %s %s\n   EN=%d   ENDESC=%s\n;", sid, nowStr(), ctag, compl, en, desc)
}

// compldFrame 成功操作响应。
func compldFrame(ctag string) string { return opFrame(ctag, "COMPLD", 0, "成功。") }

// denyFrame 拒绝响应。
func denyFrame(ctag string, en int, desc string) string { return opFrame(ctag, "DENY", en, desc) }

// delayFrame 延迟响应帧,无 EN 行。
func delayFrame(ctag string) string {
	return fmt.Sprintf("   %s %s\nM  %s DELAY\n;", sid, nowStr(), ctag)
}

// tableFrame 查询类响应帧:Title= 行 + TAB 分隔 attrib 行 + --- 分隔 + TAB 对齐 value 行。
// 单块响应,block_records 即行数;attrs 与每行 values 逐位对齐。
func tableFrame(ctag, title string, attrs []string, rows [][]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "   %s %s\n", sid, nowStr())
	fmt.Fprintf(&b, "M  %s COMPLD\n", ctag)
	b.WriteString("   EN=0   ENDESC=成功。\n")
	b.WriteString("   total_blocks=1\n   block_number=1\n")
	fmt.Fprintf(&b, "   block_records=%d\n\n", len(rows))
	fmt.Fprintf(&b, "Title = %s\n", title)
	b.WriteString(strings.Join(attrs, "\t"))
	b.WriteString("\n-----\n")
	for _, r := range rows {
		b.WriteString(strings.Join(r, "\t"))
		b.WriteString("\n")
	}
	b.WriteString("-----\n;\n")
	return b.String()
}

// 查询类 handler:出表格式对齐 PDF §13.1.2/§13.2.1/§15.4.4 示例。
// 无效值一律回填 "--",防 TL1 表格按空白分词时列错位。

// lstONU 查 ONU 设备信息(PDF §13.1.2)。
func (st *state) lstONU(cmd cmdReq) string {
	attrs := []string{"OLTID", "PONID", "ONUNO", "NAME", "DESC", "ONUTYPE", "IP", "AUTHTYPE", "MAC", "LOID"}
	var rows [][]string
	for _, o := range st.onus {
		if !matchONU(o, cmd) {
			continue
		}
		loid := "--"
		if o.authtype == "LOID" || o.authtype == "LOIDONCEON" {
			loid = o.onuid
		}
		rows = append(rows, []string{
			o.oltid, o.ponid, strconv.Itoa(o.onuno), "--", nz(o.desc), nz(o.onutype),
			"--", o.authtype, "--", loid,
		})
	}
	return tableFrame(cmd.ctag, "LST ONU information", attrs, rows)
}

// lstPONVLAN 查 PON 口 VLAN/业务流(PDF §13.2.1);单层业务 SVLAN 回 "--"。
func (st *state) lstPONVLAN(cmd cmdReq) string {
	attrs := []string{"OLTID", "PONID", "ONUNUMBER", "AUTHTYPE", "AUTHCODE", "SVLAN", "CVLAN", "UV", "CID", "DESC"}
	var rows [][]string
	for _, s := range st.svcs {
		if !matchSvc(s, cmd) {
			continue
		}
		sv := "--"
		if s.hasSVLAN {
			sv = strconv.Itoa(s.svlan)
		}
		rows = append(rows, []string{
			s.oltid, s.ponid, st.onuNoOf(s.oltid, s.ponid, s.onuid), "LOID", s.onuid,
			sv, strconv.Itoa(s.cvlan), strconv.Itoa(s.uv), "--", nz(s.desc),
		})
	}
	return tableFrame(cmd.ctag, "list of Pon Vlan configuration", attrs, rows)
}

// lstONUState 查 ONU 状态(PDF §15.4.4);SHOWOPTION 含 CFGSTAT 时追加该列。
func (st *state) lstONUState(cmd cmdReq) string {
	attrs := []string{"ONUID", "AdminState", "OperState", "AUTH", "AUTHINFO", "ONUIP", "LASTOFFTIME"}
	showCFGSTAT := strings.Contains(cmd.payload["SHOWOPTION"], "CFGSTAT")
	if showCFGSTAT {
		attrs = append(attrs, "CFGSTAT")
	}
	var rows [][]string
	for _, o := range st.onus {
		if !matchONU(o, cmd) {
			continue
		}
		row := []string{
			strconv.Itoa(o.onuno), o.adminState, o.operState, o.authtype, o.onuid, "--", "--",
		}
		if showCFGSTAT {
			row = append(row, o.cfgstat)
		}
		rows = append(rows, row)
	}
	return tableFrame(cmd.ctag, "list of ONU state", attrs, rows)
}

// matchONU 按 access(及兼容位 payload)过滤:给出即比对,空则通配。
func matchONU(o *onuRec, cmd cmdReq) bool {
	if v := cmd.access["OLTID"]; v != "" && v != o.oltid {
		return false
	}
	if v := cmd.access["PONID"]; v != "" && v != o.ponid {
		return false
	}
	if v := firstNonEmpty(cmd.payload["ONUID"], cmd.access["ONUID"]); v != "" && v != o.onuid {
		return false
	}
	return true
}

// matchSvc 业务流行过滤,同 matchONU 语义。
func matchSvc(s *svcRec, cmd cmdReq) bool {
	if v := cmd.access["OLTID"]; v != "" && v != s.oltid {
		return false
	}
	if v := cmd.access["PONID"]; v != "" && v != s.ponid {
		return false
	}
	if v := cmd.access["ONUID"]; v != "" && v != s.onuid {
		return false
	}
	return true
}

// onuNoOf 查业务流归属 ONU 的授权号,无卡回 "--"。
func (st *state) onuNoOf(oltid, ponid, onuid string) string {
	if o := st.onus[[3]string{oltid, ponid, onuid}]; o != nil {
		return strconv.Itoa(o.onuno)
	}
	return "--"
}

// nz 空值回 "--"(PDF: 返回 "--" 表示该属性无效)。
func nz(s string) string {
	if s == "" {
		return "--"
	}
	return s
}
