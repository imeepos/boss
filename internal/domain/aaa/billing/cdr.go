// Package billing 话单入口:AAA 从 RADIUS Accounting-Request 产出话单,
// 投递 Kafka 供 Flink 实时处理 + Doris 入库(技术栈方案 2.2 goRTR→Kafka→Flink→Doris)。
package billing

import "context"

// CDR 话单(Charge Detail Record)最小字段集。
// 完整字段随阶段7接入 OLT 侧补充,此处固化跨域契约与投递语义。
type CDR struct {
	LOID         string
	Username     string
	AcctStatus   int // rfc2866 AcctStatusType:1 Start / 2 Stop / 3 Interim
	SessionID    string
	SessionTime  uint32 // 秒
	InputOctets  uint64
	OutputOctets uint64
	NASIP        string
}

// Emitter 话单投递口:发送到 Kafka 话单 topic。
// 阶段7落地实现用 Kafka producer;此处为域边界接口,禁止混入协议层依赖。
type Emitter interface {
	Emit(ctx context.Context, cdr CDR) error
}
