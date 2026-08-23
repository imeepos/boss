// openPlatAppCreateReq validate() 回归测试:
// 修复 admin 创建开放应用报"参数非法"的根因(name 缺失/全空白未在反序列化阶段拒,
// 且 handler 用 req.validate 方法值绑定到零值 receiver,导致 binding 后
// 校验的对象不是已填充的 req)。
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

// TestOpenPlatAppCreateMethodValueCapturesZeroReceiver 钉死 root cause:
// req.validate 是值 receiver 方法值,创建时复制当时的 req(零值);
// handler 必须在 BindAndValidate 调用处用闭包 func() error { return req.validate() }
// 而不是直接传 req.validate,后者捕获的是 binding 前的零值 req。
func TestOpenPlatAppCreateMethodValueCapturesZeroReceiver(t *testing.T) {
	var req openPlatAppCreateReq
	bad := req.validate // 值 receiver 方法值,绑定到零值副本
	req.Name = "late"
	if err := bad(); err == nil {
		t.Fatal("方法值应仍看到零值 Name,但通过了校验(说明 bug 已不存在)")
	}
	good := func() error { return req.validate() } // 闭包,延迟到调用时 deref
	if err := good(); err != nil {
		t.Fatalf("闭包应看到已填充的 Name,但报: %v", err)
	}
}