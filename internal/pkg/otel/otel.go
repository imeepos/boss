// Package otel OpenTelemetry 装配:OTLP gRPC 导出(Jaeger :4317 对外映射 16831)。
// 应用侧只依赖全局 TracerProvider;中间件/业务代码经 otel.Tracer 取 tracer。
package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config 装配参数。
type Config struct {
	Endpoint string // OTLP gRPC host:port;空则降级 noop
	Service  string // service.name 资源属性
}

// Setup 初始化全局 TracerProvider,返回 shutdown(进程退出时调用以冲刷缓冲 span)。
func Setup(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.Endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(), // 内网明文,证书终止在接入层
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(cfg.Service),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
