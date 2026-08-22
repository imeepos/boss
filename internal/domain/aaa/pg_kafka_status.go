package aaa

// 话单 Kafka 投递状态(迁移 000109,Q2 话单补偿)。
// PG(cdrs)是权威侧;实时链路(Kafka→Flink)失败仅置 FAILED,
// 补偿循环从本表按 id 升序限速补投,最终一致。

import (
	"context"
	"fmt"
)

// Kafka 投递状态枚举(terms.md §4 激活回调之外的新增枚举,仅内部补偿用)。
const (
	CdrKafkaPending = "PENDING" // 未投或待补投
	CdrKafkaSent    = "SENT"    // 实时或补偿已送达
	CdrKafkaFailed  = "FAILED"  // 最近一次投递失败
)

// ListUnsentCdrs 取未送达话单(kafka_status <> SENT;id 升序,limit 限速防首轮重放洪峰)。
func (s *PGStore) ListUnsentCdrs(ctx context.Context, limit int) ([]CdrRecord, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+cdrCols+` FROM cdrs WHERE kafka_status <> 'SENT' ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("aaa: list cdrs by kafka status: %w", err)
	}
	defer rows.Close()
	out := make([]CdrRecord, 0)
	for rows.Next() {
		var c CdrRecord
		if err := rows.Scan(&c.ID, &c.Loid, &c.Username, &c.AcctStatus, &c.SessionID,
			&c.SessionTime, &c.InputOctets, &c.OutputOctets, &c.NasIP, &c.BillingStatus, &c.StartedAt); err != nil {
			return nil, fmt.Errorf("aaa: scan cdr kafka status: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// MarkCdrsKafkaStatus 批量回写 Kafka 投递状态;ids 空为 no-op。
func (s *PGStore) MarkCdrsKafkaStatus(ctx context.Context, ids []int64, status string) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE cdrs SET kafka_status = $2 WHERE id = ANY($1)`, ids, status); err != nil {
		return fmt.Errorf("aaa: mark cdrs kafka status: %w", err)
	}
	return nil
}
