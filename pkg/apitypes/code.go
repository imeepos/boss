// Package apitypes 跨服务公共 DTO 与错误码。
// 边界约定(见 docs/architecture-review.md 发现 3.1):
//   - 本包只承载「跨进程契约」的稳定类型:错误码、gRPC 消息、事件 envelope。
//   - 进程内的 DTO / 值对象 / 请求响应一律放 internal,不得放本包,
//     避免 internal 依赖对外公开库、造成未来拆分时的循环困惑。
package apitypes

// Code 统一错误码:各进程/网关/前端对齐语义,不依赖 HTTP 状态码的逐层文件约定。
type Code int32

const (
	CodeOK            Code = 0
	CodeInternal      Code = 50000
	CodeUnauthorized  Code = 40100
	CodeForbidden     Code = 40300
	CodeNotFound      Code = 40400
	CodeConflict      Code = 40900
	CodeInvalidParam  Code = 42200
	CodeResourceBusy  Code = 42300 // 端口预占冲突等
	CodeStateInvalid  Code = 40910 // 状态机非法流转
	CodeScanMismatch  Code = 40920 // 四码合一扫码不一致
	CodeDownstreamErr Code = 50200
)

// Message 预置的、可被网关直接透出的错误文案;业务域可追加更具体 detail。
func (c Code) Message() string {
	switch c {
	case CodeOK:
		return "ok"
	case CodeUnauthorized:
		return "未认证或凭证无效"
	case CodeForbidden:
		return "无权限"
	case CodeNotFound:
		return "资源不存在"
	case CodeConflict:
		return "资源冲突"
	case CodeInvalidParam:
		return "参数非法"
	case CodeResourceBusy:
		return "资源已被占用"
	case CodeStateInvalid:
		return "非法状态流转"
	case CodeScanMismatch:
		return "扫码与预绑定不一致"
	case CodeDownstreamErr:
		return "下游服务异常"
	default:
		return "内部错误"
	}
}
