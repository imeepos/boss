package adminapi

// realid-config 字段定义:实名二要素自动核验通道(阿里云实人认证 Id2MetaVerify)。
// 与 authconfig/smsconfig 同一套 biz_params 存储/secretbox 加密/掩码回显约定。

// realidFields 全部字段;仅一个 group: channel。
var realidFields = []authField{
	{Key: "realid.enabled", Group: "channel", Default: "true"},
	{Key: "realid.provider", Group: "channel", Default: "aliyun_cloudauth"},
	{Key: "realid.accessKeyId", Group: "channel"},
	{Key: "realid.accessKeySecret", Group: "channel", Secret: true},
	{Key: "realid.endpoint", Group: "channel"}, // 空 = cloudauth.aliyuncs.com
}

// realidFieldByGroup 组内字段。
func realidFieldByGroup(group string) []authField {
	out := make([]authField, 0, len(realidFields))
	for _, f := range realidFields {
		if f.Group == group {
			out = append(out, f)
		}
	}
	return out
}

// realidFieldByKey 全字段索引。
func realidFieldByKey(key string) (authField, bool) {
	for _, f := range realidFields {
		if f.Key == key {
			return f, true
		}
	}
	return authField{}, false
}

// realidTestRequired channel 组自检必填(草稿合并后校验)。
var realidTestRequired = []string{"realid.accessKeyId", "realid.accessKeySecret"}
