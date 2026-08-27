// Package buildinfo 编译期注入的构建元数据(经 -ldflags -X 填充)。
//
// LicensePublicKeyHex 是 release-platform 离线授权签发公钥(hex ed25519)。
// 生产构建必须注入(见 Makefile inject-license-key/scripts/build-server.sh);
// 未注入 = 开发构建,授权门禁降级为"未启用 + ALERT 日志"(B 档裁定:
// 公钥内嵌使"替换公钥绕过验签"需要重新编译,而非改配置文件)。
package buildinfo

// LicensePublicKeyHex release-platform 签发公钥(hex)。构建期注入:
//
//	go build -ldflags "-X github.com/ymm-001/boss/internal/pkg/buildinfo.LicensePublicKeyHex=<hex>" ./cmd/server
var LicensePublicKeyHex string