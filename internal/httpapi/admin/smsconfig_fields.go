package adminapi

// sms-config 字段定义:短信验证码通道(阿里云国际短信)与双语文案模板。
// 与 authconfig 同一套 biz_params 存储/secretbox 加密/掩码回显约定。

// smsFields 全部字段;groups: channel(通道)/template(文案)。
var smsFields = []authField{
	// 通道:阿里云国际短信
	{Key: "sms.enabled", Group: "channel", Default: "true"},
	{Key: "sms.provider", Group: "channel", Default: "aliyun_intl"},
	{Key: "sms.accessKeyId", Group: "channel"},
	{Key: "sms.accessKeySecret", Group: "channel", Secret: true},
	{Key: "sms.from", Group: "channel"}, // 阿里云国际 SenderID

	// 文案模板:{code} 占位,按区号 86/60 取用
	{Key: "sms.template.cn", Group: "template"},
	{Key: "sms.template.my", Group: "template"},
}

// smsFieldByGroup 组内字段(channel/template)。
func smsFieldByGroup(group string) []authField {
	out := make([]authField, 0, len(smsFields))
	for _, f := range smsFields {
		if f.Group == group {
			out = append(out, f)
		}
	}
	return out
}

// smsFieldByKey 全字段索引。
func smsFieldByKey(key string) (authField, bool) {
	for _, f := range smsFields {
		if f.Key == key {
			return f, true
		}
	}
	return authField{}, false
}

// smsTestRequired channel 组自检必填(草稿合并后校验)。
var smsTestRequired = []string{"sms.accessKeyId", "sms.accessKeySecret"}
