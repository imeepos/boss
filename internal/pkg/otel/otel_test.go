package otel

// OTel 装配单测:空端点降级 noop;带端点可 Shutdown。

import (
	"context"
	"testing"
)

func TestSetupNoopWhenEndpointEmpty(t *testing.T) {
	shutdown, err := Setup(context.Background(), Config{Endpoint: "", Service: "boss-server"})
	if err != nil {
		t.Fatal(err)
	}
	if shutdown == nil {
		t.Fatal("shutdown nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSetupWithEndpoint(t *testing.T) {
	// 导出器懒连接:端点无需可达,Setup 只装配 Provider。
	shutdown, err := Setup(context.Background(), Config{Endpoint: "127.0.0.1:14317", Service: "boss-test"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shutdown(context.Background()) }()
	// Shutdown 对不可达端点批量导出器只报错不 panic。
}
