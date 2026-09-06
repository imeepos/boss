// 资产身份三要素与 EPC 校验(P3-T2,迁移 000188)。
// SN/MAC/LOID 是电信 ONU 网络鉴权标识,全网唯一由 DB 部分唯一索引硬保证(uq_assets_sn/
// mac/loid,空值空串不占名额);EPC Gen2 96-bit 标准表示为 24 位 hex,头部字节限定
// SGTIN-96(0x30)/GRAI-96(0x32)/GIAI-96(0x33)/GID-96(0x35)。存量行不回填不强制,
// 校验仅作用于新写入(docs/design/asset-tag-p3-plan.md §T2)。

package asset

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrInvalidMAC MAC 格式非法(六组十六进制,冒号或横杠分隔均可)。
var ErrInvalidMAC = errors.New("asset: invalid mac format")

// ErrInvalidEPC EPC 非法(须 24 位 hex,头部字节在 30/32/33/35 内)。
var ErrInvalidEPC = errors.New("asset: invalid epc code")

// ErrAssetIdentityDuplicate 身份列唯一冲突(409 语义,message 含冲突字段名)。
type ErrAssetIdentityDuplicate struct{ Field string }

func (e *ErrAssetIdentityDuplicate) Error() string {
	return "asset: " + e.Field + " already exists"
}

// macPattern 六组十六进制,分隔符冒号或横杠(整串一致;入库原样不做规范化,
// 规范化存储与模糊匹配是计划明示的后续波次)。
var macPattern = regexp.MustCompile("^([0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){5}|[0-9a-fA-F]{2}(-[0-9a-fA-F]{2}){5})$")

// epcPattern 24 位十六进制(大小写在归一大写后校验)。
var epcPattern = regexp.MustCompile("^[0-9A-F]{24}$")

// NormalizeIdentity 归一身份三字段:SN/LOID 去首尾空格;MAC 去空格后校验格式
// (入库原样);空串一律归空(存储层 strOrNil 转 NULL,不占唯一名额)。
// MAC 非法返回 ErrInvalidMAC(42200)。
func NormalizeIdentity(sn, mac, loid string) (string, string, string, error) {
	sn = strings.TrimSpace(sn)
	loid = strings.TrimSpace(loid)
	mac = strings.TrimSpace(mac)
	if mac != "" && !macPattern.MatchString(mac) {
		return "", "", "", fmt.Errorf("asset: mac %q: %w", mac, ErrInvalidMAC)
	}
	return sn, mac, loid, nil
}

// NormalizeEPC 校验并归一 EPC:24 位 hex 大小写不敏感,入库统一大写;
// 首字节(前两字符)不在 30/32/33/35 内拒绝(ErrInvalidEPC,42200)。
func NormalizeEPC(epc string) (string, error) {
	v := strings.ToUpper(strings.TrimSpace(epc))
	if !epcPattern.MatchString(v) {
		return "", fmt.Errorf("asset: epc %q not 24-hex: %w", epc, ErrInvalidEPC)
	}
	switch v[:2] {
	case "30", "32", "33", "35":
		return v, nil
	}
	return "", fmt.Errorf("asset: epc %q head byte not in 30/32/33/35: %w", epc, ErrInvalidEPC)
}

// IdentityEffText 编辑归一:指针 nil=未传保持现值;非 nil 去首尾空格生效
// (空串即清除,落库转 NULL)。
func IdentityEffText(cur string, p *string) string {
	if p == nil {
		return cur
	}
	return strings.TrimSpace(*p)
}

// IdentityEffMac 编辑归一(MAC):nil=保持现值;非 nil 校验格式后生效
// (入库原样,空串清除)。非法返回 ErrInvalidMAC(42200)。
func IdentityEffMac(cur string, p *string) (string, error) {
	if p == nil {
		return cur, nil
	}
	v := strings.TrimSpace(*p)
	if v != "" && !macPattern.MatchString(v) {
		return "", fmt.Errorf("asset: mac %q: %w", v, ErrInvalidMAC)
	}
	return v, nil
}

// strOrNil 空串归一 NULL(身份列部分唯一索引 WHERE 非空非空串的存储口径)。
func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}
