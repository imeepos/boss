// openPlatAppCreateReq validate() 回归测试:
// 修复 admin 创建开放应用报"参数非法"的根因(name 缺失/全空白未在反序列化阶段拒)。
package adminapi

import (
	"strings"
	"testing"
)

// TestOpenPlatAppCreateReqValidate 覆盖修复后的校验语义。
func TestOpenPlatAppCreateReqValidate(t *testing.T) {
	cases := []struct {
		name   string
		req    openPlatAppCreateReq
		wantOK bool
	}{
		{name: "正常名称", req: openPlatAppCreateReq{Name: "demo-app", RateLimitRPM: 60, DailyQuota: 10000}, wantOK: true},
		{name: "空字符串", req: openPlatAppCreateReq{Name: "", RateLimitRPM: 60, DailyQuota: 10000}, wantOK: false},
		{name: "全空白", req: openPlatAppCreateReq{Name: "   ", RateLimitRPM: 60, DailyQuota: 10000}, wantOK: false},
		{name: "首尾空白但非空", req: openPlatAppCreateReq{Name: "  demo  ", RateLimitRPM: 60, DailyQuota: 10000}, wantOK: true},
		{name: "超长", req: openPlatAppCreateReq{Name: strings.Repeat("x", 65), RateLimitRPM: 60, DailyQuota: 10000}, wantOK: false},
		{name: "rpm 负数", req: openPlatAppCreateReq{Name: "demo", RateLimitRPM: -1, DailyQuota: 10000}, wantOK: false},
		{name: "quota 负数", req: openPlatAppCreateReq{Name: "demo", RateLimitRPM: 60, DailyQuota: -100}, wantOK: false},
		{name: "rpm 0 合法", req: openPlatAppCreateReq{Name: "demo", RateLimitRPM: 0, DailyQuota: 0}, wantOK: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.validate()
			if tc.wantOK && err != nil {
				t.Fatalf("期望通过但报错: %v", err)
			}
			if !tc.wantOK && err == nil {
				t.Fatal("期望报错但通过")
			}
		})
	}
}