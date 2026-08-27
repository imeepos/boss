// DB 动态配置里的密文密钥解密统一入口。
package app

import (
	"log"
	"sync"

	"github.com/ymm-001/boss/internal/pkg/secretbox"
)

// configDecryptWarned 按 field 记录已告警状态:配置热重载(60s 级)下
// 同一密钥失效每分钟刷一遍日志等于告警风暴,按字段只报第一次即可定位问题。
var configDecryptWarned sync.Map

// decryptConfigSecret 解 biz_params 里 enc:v1: 形态的密文配置值。
// 解密失败(密钥轮换后旧密文、部署环境变量丢失等)必须显性化:
// 静默返回空串会让 Stripe/SMS/MinIO/Push 实名认证等通道表现为"未配置",
// 直到业务请求撞上诡异的下游错误才被发现。这里返回空串保持原语义
// (env 兜底继续生效),但留下可 grep 的告警指明具体字段与原因。
func decryptConfigSecret(field, stored string) string {
	if stored == "" {
		return ""
	}
	plain, err := secretbox.Open(stored)
	if err == nil {
		return plain
	}
	if _, dup := configDecryptWarned.LoadOrStore(field, true); !dup {
		log.Printf("[config-secrets] DECRYPT FAILED field=%s err=%v "+
			"(密钥可能已轮换或 BOSS_AUTH_SECRET_KEY/BOSS_JWT_SECRET 变更;该字段回退 env 兜底,重新保存一次可重加密)", field, err)
	}
	return ""
}
