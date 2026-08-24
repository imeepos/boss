package adminapi

// sms-config 辅助:参数合并、完整性校验结果、合并配置 → 通道实例、测试文案。

import (
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
	"github.com/ymm-001/boss/internal/pkg/sms"
)

// sealParamSecret secret 字段加密落库(与 authconfig 同约定)。
func sealParamSecret(raw string) (string, error) { return secretbox.Seal(raw) }

// smsKeysInGroup 全部 key 属于该组才放行。
func smsKeysInGroup(values map[string]string, group string) bool {
	for key := range values {
		f, ok := smsFieldByKey(key)
		if !ok || f.Group != group {
			return false
		}
	}
	return true
}

// smsMergedParams 已存 biz_params(含默认值)叠加草稿 secret 解密后合并。
func smsMergedParams(c *gin.Context, a *app.Application, draft map[string]string, group string) (map[string]string, error) {
	list, err := a.User.ListParams(c.Request.Context())
	if err != nil {
		return nil, err
	}
	cur := map[string]string{}
	for _, p := range list {
		cur[p.Key] = p.Value
	}
	for _, f := range smsFields {
		if cur[f.Key] == "" {
			cur[f.Key] = f.Default
		}
		if f.Secret && cur[f.Key] != "" {
			if plain, err := secretbox.Open(cur[f.Key]); err == nil {
				cur[f.Key] = plain
			}
		}
	}
	for k, v := range draft {
		if f, ok := smsFieldByKey(k); ok && f.Group == group && !(f.Secret && v == "") {
			cur[k] = v
		}
	}
	return cur, nil
}

// smsCompletenessResult 配置完整性:未启用视为通过;缺必填返回明细。
func smsCompletenessResult(cur map[string]string) gin.H {
	if cur["sms.enabled"] == "false" {
		return gin.H{"ok": true, "latencyMs": 0, "message": "通道未启用,跳过自检"}
	}
	var missing []string
	for _, key := range smsTestRequired {
		if cur[key] == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return gin.H{"ok": false, "latencyMs": 0, "message": "缺少必填字段: " + joinKeys(missing)}
	}
	return gin.H{"ok": true, "latencyMs": 0, "message": "配置完整性校验通过"}
}

// smsSenderFromParams 合并配置 → 通道实例(试发用;未配置凭据时为日志通道)。
func smsSenderFromParams(cur map[string]string) sms.Sender {
	codes := map[string]string{}
	if v := cur["sms.contentCode.cn"]; v != "" {
		codes["86"] = v
	}
	if v := cur["sms.contentCode.my"]; v != "" {
		codes["60"] = v
	}
	return sms.NewAliyunIntl(sms.AliyunIntlConfig{
		AccessKeyID:     cur["sms.accessKeyId"],
		AccessKeySecret: cur["sms.accessKeySecret"],
		ContentCodes:    codes,
	})
}

// smsTestMessage 试发结果文案。
func smsTestMessage(err error) string {
	if err == nil {
		return "测试短信已提交发送"
	}
	if errors.Is(err, sms.ErrUnsupportedRegion) {
		return "号码区号不支持(仅 +86 / +60)"
	}
	if errors.Is(err, sms.ErrContentCodeMissing) {
		return "该区号未配置报备模板 ContentCode(短信配置页-报备模板)"
	}
	return "发送失败: " + err.Error()
}

// randDigitsStr n 位随机数字串(测试码)。
func randDigitsStr(n int) (string, error) {
	out := make([]byte, n)
	for i := range out {
		v, err := cryptoRandInt(10)
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + v)
	}
	return string(out), nil
}

// cryptoRandInt [0,max) 密码学随机整数。
func cryptoRandInt(max int64) (int64, error) {
	v, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return v.Int64(), nil
}
