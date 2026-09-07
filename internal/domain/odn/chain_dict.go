package odn

// W3 资源链枚举字典与归一化辅助(说明页枚举 sheet 为权威校验字典)。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// validateChainEnums 枚举列映射(③留空=NULL,未知值拒绝);分光比经枚举字典解析。
func validateChainEnums(r *ResourceChainRow, rec *ChainRecord) string {
	must := func(label, v string, dict map[string]string, dst *string) string {
		if v == "" {
			return ""
		}
		code, ok := dict[v]
		if !ok {
			return label + "非法: " + v
		}
		*dst = code
		return ""
	}
	checks := []struct {
		label, v string
		dict     map[string]string
		dst      *string
	}{
		{"端口状态", r.PortStatus, chainPortStatus, &rec.PortStatus},
		{"敷设方式", r.LayingMethod, chainLayingMethod, &rec.LayingMethod},
		{"ROW状态", r.RowStatus, chainRowStatus, &rec.RowStatus},
		{"PECE状态", r.PeceStatus, chainPeceStatus, &rec.PeceStatus},
	}
	for _, c := range checks {
		if reason := must(c.label, c.v, c.dict, c.dst); reason != "" {
			return reason
		}
	}
	var reason string
	if rec.Split1, reason = parseChainRatio(r.Split1Ratio, "一级分光比"); reason != "" {
		return reason
	}
	if rec.Split2, reason = parseChainRatio(r.Split2Ratio, "二级分光比"); reason != "" {
		return reason
	}
	rec.OdfPort, rec.Split1Port, rec.Split2Port = r.OdfPort, r.Split1Port, r.Split2Port
	rec.FiberCode, rec.FrTo, rec.Remark = r.FiberCode, r.FrTo, r.Remark
	return ""
}

// validateChainCodes 箱体编码校验:格式(规范 5.1)+ 列前缀一致(编码即信息);ODF 编号原样保留
// (站点前缀引用如 SITE001_ODF001_A 不展开为设备,展开与否由存储层按规范格式判定,不猜填)。
func validateChainCodes(r *ResourceChainRow, rec *ChainRecord) string {
	cols := []struct {
		label, code, kind string
		dst               *string
	}{
		{"OCC编号", r.OccCode, DevOCC, &rec.OccCode},
		{"ODB编号", r.OdbCode, DevODB, &rec.OdbCode},
		{"OBD编号", r.ObdCode, DevOBD, &rec.ObdCode},
		{"SDB编号", r.SdbCode, DevSDB, &rec.SdbCode},
		{"SBD编号", r.SbdCode, DevSBD, &rec.SbdCode},
	}
	for _, c := range cols {
		if c.code == "" {
			continue
		}
		if reason := chainCodeReason(c.kind, c.code); reason != "" {
			return c.label + reason
		}
		*c.dst = c.code
	}
	rec.OdfCode = r.OdfCode
	return ""
}

// chainCodeReason 设备码校验(格式+前缀一致);返回原因后缀(空=通过)。
func chainCodeReason(kind, code string) string {
	got, err := ValidateDeviceCode(code)
	if err != nil {
		return " 编码格式非法: " + code
	}
	if got != kind {
		return fmt.Sprintf(" 编码前缀须为 %s: %s", kind, code)
	}
	return ""
}

// parseChainRatio 分光比枚举解析("1:8"→8;留空=0;未知值拒绝)。
func parseChainRatio(v, label string) (int, string) {
	if v == "" {
		return 0, ""
	}
	n, ok := splitRatioEnum[v]
	if !ok {
		return 0, label + "非法: " + v
	}
	return n, ""
}

// chainFingerprint 规则⑥:23 列归一化指纹(全等行=同一资源关系)。
func chainFingerprint(r *ChainRecord) string {
	parts := []string{
		r.Lifecycle, r.SiteCode, r.SiteName, r.OltCode,
		r.OdfCode, r.OdfPort, r.OccCode, r.OdbCode, r.ObdCode,
		strconv.Itoa(r.Split1), r.Split1Port, r.SdbCode, r.SbdCode,
		strconv.Itoa(r.Split2), r.Split2Port, strconv.Itoa(r.TotalSplit),
		r.FiberCode, r.FrTo, r.PortStatus, r.LayingMethod, r.RowStatus,
		r.PeceStatus, r.Remark,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, string(rune(31)))))
	return hex.EncodeToString(sum[:])
}

// trimRow 去除全部字段空白(手填模板常见尾随空格,防指纹误判重复)。
func trimRow(r *ResourceChainRow) {
	r.ResourceStatus = strings.TrimSpace(r.ResourceStatus)
	r.SiteCode = strings.TrimSpace(r.SiteCode)
	r.SiteName = strings.TrimSpace(r.SiteName)
	r.OltCode = strings.TrimSpace(r.OltCode)
	r.OdfCode = strings.TrimSpace(r.OdfCode)
	r.OdfPort = strings.TrimSpace(r.OdfPort)
	r.OccCode = strings.TrimSpace(r.OccCode)
	r.OdbCode = strings.TrimSpace(r.OdbCode)
	r.ObdCode = strings.TrimSpace(r.ObdCode)
	r.Split1Ratio = strings.TrimSpace(r.Split1Ratio)
	r.Split1Port = strings.TrimSpace(r.Split1Port)
	r.SdbCode = strings.TrimSpace(r.SdbCode)
	r.SbdCode = strings.TrimSpace(r.SbdCode)
	r.Split2Ratio = strings.TrimSpace(r.Split2Ratio)
	r.Split2Port = strings.TrimSpace(r.Split2Port)
	r.TotalSplit = strings.TrimSpace(r.TotalSplit)
	r.FiberCode = strings.TrimSpace(r.FiberCode)
	r.FrTo = strings.TrimSpace(r.FrTo)
	r.PortStatus = strings.TrimSpace(r.PortStatus)
	r.LayingMethod = strings.TrimSpace(r.LayingMethod)
	r.RowStatus = strings.TrimSpace(r.RowStatus)
	r.PeceStatus = strings.TrimSpace(r.PeceStatus)
	r.Remark = strings.TrimSpace(r.Remark)
}
