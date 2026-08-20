package adminapi

// auth-config 字段定义:与 boss/designs/auth-config-v1.spec.md 的字段 key 一一对应。
// secret 字段经 secretbox 加密落 biz_params;GET 只回掩码标记(hasValue),PUT 空串表示不修改。

// authField 单个配置字段元数据。
type authField struct {
	Key     string // biz_params key,如 auth.cn.appKey
	Group   string // cn / my / fallback
	Secret  bool   // 加密存储 + 掩码回显
	Default string // biz_params 无值时的缺省
}

// authFields 全部字段(spec §1.3–§1.5);fallback 组含 compliance 键。
var authFields = []authField{
	// 中国区 · 极光一键登录
	{Key: "auth.cn.enabled", Group: "cn", Default: "false"},
	{Key: "auth.cn.appKey", Group: "cn"},
	{Key: "auth.cn.appSecret", Group: "cn", Secret: true},
	{Key: "auth.cn.packageName", Group: "cn"},
	{Key: "auth.cn.preloadTimeoutMs", Group: "cn", Default: "5000"},

	// 马来西亚/海外 · 号码认证
	{Key: "auth.my.enabled", Group: "my", Default: "false"},
	{Key: "auth.my.provider", Group: "my", Default: "none"},
	{Key: "auth.my.smsProvider", Group: "my", Default: "engagelab"},
	{Key: "auth.my.apiKey", Group: "my", Secret: true},
	{Key: "auth.my.countryCode", Group: "my", Default: "+60"},
	{Key: "auth.my.smsSign", Group: "my"},

	// 降级与合规
	{Key: "auth.fallback.smsOnFail", Group: "fallback", Default: "true"},
	{Key: "auth.fallback.billingAlert", Group: "fallback", Default: "true"},
	{Key: "auth.fallback.autoRegister", Group: "fallback", Default: "false"},
	{Key: "auth.compliance.privacyVersion", Group: "fallback"},
	{Key: "auth.compliance.agreementUrl", Group: "fallback"},
}

// authFieldByGroup 组内字段(cn/my/fallback)。
func authFieldByGroup(group string) []authField {
	out := make([]authField, 0, len(authFields))
	for _, f := range authFields {
		if f.Group == group {
			out = append(out, f)
		}
	}
	return out
}

// authFieldByKey 全字段索引。
func authFieldByKey(key string) (authField, bool) {
	for _, f := range authFields {
		if f.Key == key {
			return f, true
		}
	}
	return authField{}, false
}

// authTestRequired 各组自检必填的明文字段(草稿合并后校验)。
var authTestRequired = map[string][]string{
	"cn": {"auth.cn.appKey", "auth.cn.appSecret", "auth.cn.packageName"},
	"my": {"auth.my.apiKey"},
}
