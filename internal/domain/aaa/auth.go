package aaa

import (
	"context"
	"errors"
)

// 认证失败原因码(auth_logs.fail_reason;空=SUCCESS 或存量行)。契约:docs/contract/fields.md §8A。
const (
	FailReasonBadCredential = "BAD_CREDENTIAL" // PAP/CHAP 缺失或不符;未设密且开关关
	FailReasonLocked        = "LOCKED"         // 连续失败达阈值,锁定窗口内
	FailReasonNotFound      = "NOT_FOUND"      // LOID 无对应账号
	FailReasonSuspended     = "SUSPENDED"      // 停服
	FailReasonClosed        = "CLOSED"         // 注销
)

// ErrLocked 防爆破锁定中(连续失败达阈值,窗口内一律拒绝)。
var ErrLocked = errors.New("aaa: loid locked")

// ErrBadCredential 凭据缺失或校验失败(PAP/CHAP 皆无、口令不符、未设密且开关关闭)。
var ErrBadCredential = errors.New("aaa: bad credential")

// CHAPCredentials RFC 1994 CHAP 三元组(NAS 随 Access-Request 上送)。
type CHAPCredentials struct {
	Ident     byte   // CHAP-Password(3) 首字节
	Challenge []byte // CHAP-Challenge(60);缺省=Request Authenticator(RFC 2865 §5.3)
	Response  []byte // CHAP-Password 后 16 字节 MD5 摘要
}

// Credentials Access-Request 携带的凭据;PAP 与 CHAP 皆空时必然拒绝。
type Credentials struct {
	PAP  string           // User-Password(共享密钥解密后明文)
	CHAP *CHAPCredentials // 非 nil 表示走 CHAP
}

// CredentialAuthenticator 凭据校验决策口(RADIUS 认证链路专用;错误即失败原因:
// ErrLocked/ErrNotFound/ErrSuspended/ErrClosed/ErrBadCredential)。
type CredentialAuthenticator interface {
	Authenticate(ctx context.Context, loid string, creds Credentials) (Decision, error)
}
