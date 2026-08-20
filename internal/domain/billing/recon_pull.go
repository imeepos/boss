package billing

// 渠道流水自动拉取抽象(对账自动化第一阶段)。
// 裁定 2026-08-20(sms-payment-channel):真实渠道凭据未到,不 vendor SDK;
// 本层只定契约与编排,真渠道(微信/支付宝/线下导出)后续 Register 接入。

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrChannelManual 渠道为手工模式:未配置自动拉取,流水走 POST statement 录入。
var ErrChannelManual = errors.New("billing: channel is manual-reconcile only")

// ErrChannelNotConfigured 渠道未注册任何源(拼写错误或未接入)。
var ErrChannelNotConfigured = errors.New("billing: channel source not configured")

// ChannelSource 渠道流水源:按日期拉取该渠道对账单(金额单位元,与 payments 对齐)。
type ChannelSource interface {
	PullStatement(ctx context.Context, channel string, date time.Time) ([]ChannelStatementRow, error)
}

// ChannelSourceRegistry 源注册表(线程安全,启动期注册)。
type ChannelSourceRegistry struct {
	byChannel map[string]ChannelSource
}

// NewChannelSourceRegistry 建注册表;nil 源条目表示 manual 模式。
func NewChannelSourceRegistry() *ChannelSourceRegistry {
	return &ChannelSourceRegistry{byChannel: map[string]ChannelSource{}}
}

// Register 注册渠道源;source 传 nil 表示该渠道仅手工对账。
func (r *ChannelSourceRegistry) Register(channel string, source ChannelSource) {
	r.byChannel[channel] = source
}

// source 取渠道源;未注册返回 (nil, ErrChannelNotConfigured),注册为 nil 返回 (nil, nil)。
func (r *ChannelSourceRegistry) source(channel string) (ChannelSource, error) {
	s, ok := r.byChannel[channel]
	if !ok {
		return nil, ErrChannelNotConfigured
	}
	return s, nil // s==nil 表示 manual
}

// AutoReconciler 自动对账编排:按渠道建当日批次(幂等)→拉流水→逐行比对。
type AutoReconciler struct {
	Recon   ReconService
	Sources *ChannelSourceRegistry
}

// AutoReconcileResult 单渠道编排结果。
type AutoReconcileResult struct {
	Channel string     `json:"channel"`
	BatchNo string     `json:"batchNo"`
	Status  string     `json:"status"` // DIFF_PENDING / SETTLED / MANUAL_PENDING(建批待手工录入)
	Skipped string     `json:"skipped,omitempty"` // 非空=该渠道本轮未比对,值为原因
}

// AutoReconcile 对每个渠道:当日批次已存在则复用(幂等),否则建批;
// 配置了源则拉流水并 RecordChannelStatement,manual 渠道只建批。
// 单渠道失败不中断批次循环,结果里带 skipped 原因。
func (a *AutoReconciler) AutoReconcile(ctx context.Context, date time.Time, channels []string) ([]AutoReconcileResult, error) {
	out := make([]AutoReconcileResult, 0, len(channels))
	for _, ch := range channels {
		res, err := a.reconcileChannel(ctx, date, ch)
		if err != nil {
			return out, fmt.Errorf("billing: auto reconcile %s: %w", ch, err)
		}
		out = append(out, res)
	}
	return out, nil
}

// reconcileChannel 单渠道编排。
func (a *AutoReconciler) reconcileChannel(ctx context.Context, date time.Time, channel string) (AutoReconcileResult, error) {
	res := AutoReconcileResult{Channel: channel}
	batch, err := a.ensureBatch(ctx, date, channel)
	if err != nil {
		return res, err
	}
	res.BatchNo = batch.BatchNo

	src, err := a.Sources.source(channel)
	if err != nil {
		res.Skipped = err.Error()
		return res, nil
	}
	if src == nil { // manual 模式:批次就绪,等待 POST statement
		res.Status = "MANUAL_PENDING"
		res.Skipped = ErrChannelManual.Error()
		return res, nil
	}
	rows, err := src.PullStatement(ctx, channel, date)
	if err != nil {
		res.Skipped = "pull: " + err.Error()
		return res, nil
	}
	if err := a.Recon.RecordChannelStatement(ctx, batch.ID, rows); err != nil {
		return res, err
	}
	fresh, err := a.Recon.GetReconciliation(ctx, batch.BatchNo)
	if err != nil {
		return res, err
	}
	res.Status = fresh.Status
	return res, nil
}

// ensureBatch 当日该渠道批次幂等获取:存在则复用(任意状态),否则 Append 新批。
// batch_no 规则 PC-YYYYMMDD-NN,NN 取当日已占用序号最大值+1(admin 触发,低并发可接受)。
func (a *AutoReconciler) ensureBatch(ctx context.Context, date time.Time, channel string) (*ReconBatch, error) {
	day := date.Format("20060102")
	prefix := "PC-" + day + "-"
	all, err := a.Recon.ListReconciliations(ctx)
	if err != nil {
		return nil, err
	}
	maxSeq := 0
	for i := range all {
		b := all[i]
		if b.Channel == channel && len(b.BatchNo) > len(prefix) && b.BatchNo[:len(prefix)] == prefix {
			return &b, nil // 当日已有该渠道批次,直接复用
		}
		if b.BatchNo[:min(len(b.BatchNo), len(prefix))] == prefix {
			var seq int
			if _, err := fmt.Sscanf(b.BatchNo[len(prefix):], "%d", &seq); err == nil && seq > maxSeq {
				maxSeq = seq
			}
		}
	}
	id, err := a.Recon.AppendReconciliation(ctx, ReconBatch{
		BatchNo: fmt.Sprintf("%s%02d", prefix, maxSeq+1), Channel: channel,
		ChannelAmount: 0, SystemAmount: 0, Status: "DIFF_PENDING",
	})
	if err != nil {
		return nil, err
	}
	return &ReconBatch{ID: id, BatchNo: fmt.Sprintf("%s%02d", prefix, maxSeq+1), Channel: channel}, nil
}
