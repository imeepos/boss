package tl1

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// KV 单条 k=v 参数。
type KV struct{ K, V string }

// Command 一条 TL1 命令。target 恒空,按文档占用冒号位。
type Command struct {
	Verb    string // "ADD-ONU" / "LST-ONUSTATE" ...
	Access  []KV   // OLTID=...,PONID=NA-0-7-5
	Payload []KV   // AUTHTYPE=LOID,ONUID=...
}

// Response 解析后的响应;查询类 Rows 为表格行(attrib→value),操作类 Rows 为空。
type Response struct {
	SID, CTag, Completion string // COMPLD/DELAY/DENY/PRTL/RTRV
	EN                    int
	ENDESC                string
	Rows                  []map[string]string
	Raw                   string // 原始报文,失败留痕用
}

// whitelistRe 值参数白名单字符集(OCTET STRING):字母数字 +( )_+-./ 与反斜杠。
var whitelistRe = regexp.MustCompile("^[A-Za-z0-9 ()_+./\\\\-]+$")

// Build 序列化为 "<VERB>::<K=V,K=V>:<ctag>::<K=V...>;",target 恒空留双冒号占位。
// Access 与 Payload 的键值均过白名单,违规即 error(fail fast),防脏参数入网管。
func Build(c Command, ctag string) ([]byte, error) {
	if !whitelistRe.MatchString(c.Verb) {
		return nil, fmt.Errorf("%w: verb %q", ErrBadParam, c.Verb)
	}
	acc, err := kvs(c.Access)
	if err != nil {
		return nil, err
	}
	pay, err := kvs(c.Payload)
	if err != nil {
		return nil, err
	}
	line := c.Verb + "::" + acc + ":" + ctag + "::" + pay + ";"
	return []byte(line), nil
}

// kvs 序列化参数块 "K=V,K=V"(空则留空),逐项校验白名单。
func kvs(list []KV) (string, error) {
	if len(list) == 0 {
		return "", nil
	}
	var b strings.Builder
	for i, kv := range list {
		if !whitelistRe.MatchString(kv.K) || !whitelistRe.MatchString(kv.V) {
			return "", fmt.Errorf("%w: %q=%q", ErrBadParam, kv.K, kv.V)
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(kv.K)
		b.WriteByte('=')
		b.WriteString(kv.V)
	}
	return b.String(), nil
}

// headerRe 响应头行: 缩进空格 + "HW_ip" + 日期时间。
var headerRe = regexp.MustCompile("^\\s*HW_\\S+\\s+\\d{4}-\\d{2}-\\d{2}\\s+\\d{2}:\\d{2}:\\d{2}")

// complRe 完成码行: "M  <ctag> <COMPLD|DELAY|DENY|PRTL|RTRV>"。
var complRe = regexp.MustCompile("^\\s*M\\s+(\\S+)\\s+(COMPLD|DELAY|DENY|PRTL|RTRV)\\s*$")

// enRe EN 码: 搜 "EN=<数字>"。
var enRe = regexp.MustCompile("EN\\s*=\\s*(\\d+)")

// endescRe ENDESC 文案(只存不判,定位到行尾)。
var endescRe = regexp.MustCompile("ENDESC\\s*=\\s*(.*)$")

// lineSepRe 纯横线分隔行(如 "-----")。
var lineSepRe = regexp.MustCompile("^\\s*[-_=]+\\s*$")

// titleRe "Title = list of ..." 说明行。
var titleRe = regexp.MustCompile("^Title\\s*=\\s*")

// Parse 解析完整报文(调用方保证已按终止符收全,含 > 续块拼接后的全文)。
// 跳过头行、Title 行与横线分隔行;查询表首行 attrib 列表按空白/TAB 分词,
// 后续 value 行按同序填入 Rows。; 结尾与 > 续块标记不影响表格行。
func Parse(raw string) (*Response, error) {
	resp := &Response{Raw: raw}
	if strings.TrimSpace(raw) == "" {
		return nil, ErrParse
	}
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	var st parseState
	for _, ln := range lines {
		parseLine(resp, ln, &st)
	}
	return resp, nil
}

type parseState struct {
	inTable, headRead, pendingHdr bool
	attrs                         []string
}

// parseLine 处理单行:头行/完成码/EN+ENDESC/表格行列/终止符。
func parseLine(resp *Response, ln string, st *parseState) {
	trim := strings.TrimSpace(ln)
	switch {
	case strings.HasPrefix(trim, ";") || strings.HasPrefix(trim, ">"):
		// 终止符: ; 结束, > 后续块(重置 attrib 重读)。
		st.pendingHdr = strings.HasPrefix(trim, ">")
	case headerRe.MatchString(trim) && !st.headRead:
		st.headRead = true // 头行无业务字段,仅标记
	case complRe.MatchString(trim):
		m := complRe.FindStringSubmatch(trim)
		resp.CTag, resp.Completion = m[1], m[2]
	case strings.Contains(trim, "EN=") || strings.Contains(trim, "ENDESC="):
		if m := enRe.FindStringSubmatch(trim); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				resp.EN = n
			}
		}
		if m := endescRe.FindStringSubmatch(trim); m != nil {
			resp.ENDESC = strings.TrimSpace(m[1])
		}
	case titleRe.MatchString(trim):
		st.inTable = true
	case lineSepRe.MatchString(trim) || trim == "":
		// 分隔行与空行忽略
	case st.inTable && (len(st.attrs) == 0 || st.pendingHdr):
		st.attrs = strings.Fields(ln) // attrib 首行/续块 attrib 头
		st.pendingHdr = false
	case st.inTable:
		resp.Rows = append(resp.Rows, rowOf(st.attrs, ln))
	default:
		// 非识别行忽略,不报错
	}
}

// rowOf 把 value 行按 attrib 顺序填入 map;缺列留空。
func rowOf(attrs []string, ln string) map[string]string {
	fields := strings.Fields(ln)
	row := make(map[string]string, len(attrs))
	for i, a := range attrs {
		if i < len(fields) {
			row[a] = fields[i]
		} else {
			row[a] = ""
		}
	}
	return row
}
