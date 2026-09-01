package user

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ErrPlatformMissing 平台总公司(is_platform)未配置,归属兜底链断裂(与 order 域同口径,000077)。
var ErrPlatformMissing = errors.New("user: platform legal entity missing")

// inlineSource 内联建址来源标记(migrations/000171 addresses.source)。
const inlineSource = "order-inline"

// inlineNameMax 单级地址名上限;超长属异常输入,防 path 膨胀。
const inlineNameMax = 60

// InlineAddressLevel 内联建址单级入参。
type InlineAddressLevel struct {
	Level int8   // 1市 2区 3街道 4小区 5楼栋
	Name  string // 展示名(回显靠 name,label 仅作 path 段)
}

// InlineAddressInput 内联建址入参:五级全必填,缺失层级就地补建(零阻塞)。
// CustomerID 0=未建档(代客开户场景,POST /customers/address);>0 必须已存在(开单场景)。
type InlineAddressInput struct {
	CustomerID       int64
	Levels           []InlineAddressLevel
	BackfillCustomer bool // true=回填客户档案 address_id(显式操作,非隐式同步);须 CustomerID>0
}

// InlineAddressNode 本次新建节点回执。
type InlineAddressNode struct {
	ID    int64  `json:"id"`
	Level int8   `json:"level"`
	Name  string `json:"name"`
}

// InlineAddressResult 内联建址回执:楼栋定位 + 归属预览 + 治理标记。
type InlineAddressResult struct {
	AddressID     int64               `json:"addressId"`
	FullPath      string              `json:"fullPath"`
	FullPathNames string              `json:"fullPathNames"`
	LegalEntityID int64               `json:"legalEntityId"`
	RegionPath    string              `json:"regionPath"`
	Fallback      bool                `json:"fallback"` // 祖先链无区域覆盖,兜底平台总公司
	NeedsReview   []InlineAddressNode `json:"needsReview"`
	Backfilled    bool                `json:"backfilled"`
}

// validateInlineInput 入参校验:回填须带客户;五级自上而下齐全且 level 连续。
func validateInlineInput(in InlineAddressInput) error {
	if in.CustomerID < 0 || (in.CustomerID == 0 && in.BackfillCustomer) {
		return fmt.Errorf("user: customerId %w", ErrInvalidInput)
	}
	if len(in.Levels) != 5 {
		return fmt.Errorf("user: levels need 5, got %d: %w", len(in.Levels), ErrInvalidInput)
	}
	for i, lv := range in.Levels {
		name := strings.TrimSpace(lv.Name)
		if lv.Level != int8(i+1) || name == "" || len([]rune(name)) > inlineNameMax {
			return fmt.Errorf("user: level %d invalid: %w", i+1, ErrInvalidInput)
		}
	}
	return nil
}

// addrLabelFromName name→ltree label:ASCII 词形直接转写(空格转下划线),
// 非法字符(中文等)退化为 n_+sha1 前 8 位稳定后缀;回显靠 name,客服不感知 label。
func addrLabelFromName(name string) string {
	n := strings.ToLower(strings.Join(strings.Fields(name), "_"))
	if addrLabelRe.MatchString(n) {
		if len(n) > 48 {
			n = n[:48]
		}
		return n
	}
	sum := sha1.Sum([]byte(name))
	return "n_" + hex.EncodeToString(sum[:])[:8]
}

// chainNames 摘取五级展示名(已 TrimSpace)。
func chainNames(in InlineAddressInput) []string {
	names := make([]string, len(in.Levels))
	for i, lv := range in.Levels {
		names[i] = strings.TrimSpace(lv.Name)
	}
	return names
}

// needsReviewCap 1-3 级内联新建强制复核(migrations/000171)。
func needsReviewCap(level int8) bool { return level <= 3 }
