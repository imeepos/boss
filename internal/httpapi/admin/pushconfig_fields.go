package adminapi

// push-config 字段定义:极光 JPush 聚合推送通道。
// 与 authconfig/smsconfig 同一套 biz_params 存储/secretbox 加密/掩码回显约定。

// pushFields 全部字段;groups: channel。
var pushFields = []authField{
	{Key: "push.enabled", Group: "channel", Default: "true"},
	{Key: "push.provider", Group: "channel", Default: "jpush"},
	{Key: "push.jpush.appKey", Group: "channel"},
	{Key: "push.jpush.masterSecret", Group: "channel", Secret: true},
	{Key: "push.jpush.apiUrl", Group: "channel", Default: "https://bjapi.push.jiguang.cn/v3"},
	{Key: "push.jpush.apnsProduction", Group: "channel", Default: "true"},
	{Key: "push.jpush.liveTime", Group: "channel", Default: "86400"},
}

// pushFieldByGroup 组内字段(channel)。
func pushFieldByGroup(group string) []authField {
	out := make([]authField, 0, len(pushFields))
	for _, f := range pushFields {
		if f.Group == group {
			out = append(out, f)
		}
	}
	return out
}

// pushFieldByKey 全字段索引。
func pushFieldByKey(key string) (authField, bool) {
	for _, f := range pushFields {
		if f.Key == key {
			return f, true
		}
	}
	return authField{}, false
}

// pushTestRequired channel 组自检必填(草稿合并后校验)。
var pushTestRequired = []string{"push.jpush.appKey", "push.jpush.masterSecret"}
