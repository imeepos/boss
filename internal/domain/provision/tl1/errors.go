package tl1

import "errors"

// 包级错误哨兵。业务层按哨兵分类(ErrConnBroken 触发重连、ErrAuth 不重试),
// 具体 EN/ENDESC 经 Response 返回,由调用方留痕。

var (
	// ErrConnBroken 连接断/读超时,可安全重建会话。
	ErrConnBroken = errors.New("tl1: connection broken")
	// ErrAuth 鉴权被拒(LOGIN DENY)。
	ErrAuth = errors.New("tl1: authentication failed")
	// ErrBadParam 命令参数含白名单外字符,构建时 fail fast。
	ErrBadParam = errors.New("tl1: parameter outside whitelist")
	// ErrParse 报文无法解析,原始报文留在返回的 Raw 中供留痕。
	ErrParse = errors.New("tl1: cannot parse response")
	// ErrDenied 命令被网元拒绝(DENY)。
	ErrDenied = errors.New("tl1: command denied")
	// ErrDelay 网元延迟响应,等待同 ctag 最终响应。
	ErrDelay = errors.New("tl1: delayed response")
)

// ENClass 把 U2000 错误码(EN)归为语义大类,供错误分类与日志标记。
// 具体 EN 取值随真实联调补备注,现仅留归类位。
// 分类约定: 0=成功; 鉴权类=1; 已存在类=2; 不存在类=3; 参数/语法类=4;
// 资源不足=5; 其它网元自报=6; 未知=0。占位注释,后续按 PDF EN 表校准。
func ENClass(en int) int {
	if en == 0 {
		return 0
	}
	return 6 // unknown,待联调细化
}
