package billing

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeReconSvc 最小 ReconService 桩:只驱动 AutoReconcile 编排所需路径。
type fakeReconSvc struct {
	batches []ReconBatch
	items   map[int64][]ChannelStatementRow
	nextID  int64
}

func (f *fakeReconSvc) ListReconciliations(context.Context) ([]ReconBatch, error) {
	return f.batches, nil
}

func (f *fakeReconSvc) AppendReconciliation(_ context.Context, b ReconBatch) (int64, error) {
	f.nextID++
	b.ID = f.nextID
	f.batches = append(f.batches, b)
	return b.ID, nil
}

func (f *fakeReconSvc) SettleReconciliation(context.Context, string) error { return nil }

func (f *fakeReconSvc) GetReconciliation(_ context.Context, batchNo string) (*ReconBatch, error) {
	for _, b := range f.batches {
		if b.BatchNo == batchNo {
			return &b, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeReconSvc) RecordChannelStatement(_ context.Context, batchID int64, rows []ChannelStatementRow) error {
	f.items[batchID] = rows
	return nil
}

func (f *fakeReconSvc) ListReconciliationItems(context.Context, int64) ([]ReconItem, error) {
	return nil, nil
}

// fakeSource 可编程渠道源。
type fakeSource struct {
	rows []ChannelStatementRow
	err  error
}

func (s fakeSource) PullStatement(context.Context, string, time.Time) ([]ChannelStatementRow, error) {
	return s.rows, s.err
}

// 编排:建批幂等 + 源拉取比对 + manual 渠道只建批 + 未注册渠道跳过。
func TestAutoReconcile(t *testing.T) {
	date := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	reg := NewChannelSourceRegistry()
	reg.Register("微信", fakeSource{rows: []ChannelStatementRow{{ChannelRef: "PAY-1", Amount: 88}}})
	reg.Register("线下营业厅", nil) // manual
	recon := &fakeReconSvc{items: map[int64][]ChannelStatementRow{}}
	a := &AutoReconciler{Recon: recon, Sources: reg}

	got, err := a.AutoReconcile(context.Background(), date, []string{"微信", "线下营业厅", "支付宝"})
	if err != nil {
		t.Fatalf("AutoReconcile: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("results=%+v", got)
	}
	if got[0].BatchNo != "PC-20260820-01" || got[0].Status != "DIFF_PENDING" {
		t.Fatalf("wechat=%+v", got[0])
	}
	if got[1].Status != "MANUAL_PENDING" || got[1].Skipped == "" {
		t.Fatalf("manual=%+v", got[1])
	}
	if got[2].Skipped != ErrChannelNotConfigured.Error() {
		t.Fatalf("unregistered=%+v", got[2])
	}
	if got[2].BatchNo != "PC-20260820-03" {
		t.Fatalf("unregistered batchNo=%s(未注册渠道也应建批)", got[2].BatchNo)
	}

	// 幂等:重跑不新建批次。
	before := len(recon.batches)
	if _, err := a.AutoReconcile(context.Background(), date, []string{"微信", "线下营业厅", "支付宝"}); err != nil {
		t.Fatal(err)
	}
	if len(recon.batches) != before {
		t.Fatalf("batches=%d, want %d(幂等失败)", len(recon.batches), before)
	}

	// 源报错:批次保留,skipped 带原因,整体不失败。
	reg.Register("微信", fakeSource{err: errors.New("渠道网关超时")})
	got2, err := a.AutoReconcile(context.Background(), date, []string{"微信"})
	if err != nil {
		t.Fatalf("pull 失败不应整体报错: %v", err)
	}
	if got2[0].Skipped == "" {
		t.Fatalf("skipped 为空: %+v", got2[0])
	}
}
