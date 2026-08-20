package realid

import (
	"context"
	"errors"
)

// 核验结论(对齐 verifications.result 枚举 PASS/FAIL, terms.md)。
const (
	Pass = "PASS"
	Fail = "FAIL"
)

// ErrDisabled 通道未配置凭据(提交保持 PENDING 走人工核验)。
var ErrDisabled = errors.New("realid: channel disabled")

// Verifier 实名二要素核验通道(姓名 + 身份证号 → PASS/FAIL)。
type Verifier interface {
	Verify(ctx context.Context, name, idNo string) (string, error)
}
