package quadlink

// P1-T1 资产联动单测:sink 装配后 MATCH/拆机的联动调用与强一致失败语义。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"

	"github.com/ymm-001/boss/internal/domain/asset"
)

// fakeSink 记录型联动口;deployErr 模拟资产侧失败(强一致回滚路径)。
type fakeSink struct {
	deployed  [][2]int64 // (assetID, addressID)
	released  []int64
	deployErr error
}

func (f *fakeSink) MarkDeployed(_ context.Context, _ asset.ExecQuerier, assetID, addressID, _ int64, _ string) error {
	if f.deployErr != nil {
		return f.deployErr
	}
	f.deployed = append(f.deployed, [2]int64{assetID, addressID})
	return nil
}

func (f *fakeSink) MarkReleased(_ context.Context, _ asset.ExecQuerier, assetID int64) error {
	f.released = append(f.released, assetID)
	return nil
}

// scanHappyMock 拼装一次 MATCH 扫码的最小 mock 序列(守卫空集)。
func scanHappyMock(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id = \$1`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
	mock.ExpectQuery(`FROM ports`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
	mock.ExpectQuery(`FROM quad_links`).
		WithArgs(int64(11)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(5), int64(3), int64(11), int64(21), int64(1), "主品牌·企业", "UNLINKED"))
	mock.ExpectQuery(`FROM tags`).
		WithArgs("EPC-OK").
		WillReturnRows(mock.NewRows([]string{"id", "bound_asset_id"}).AddRow(int64(9), int64(5)))
	mock.ExpectBegin()
	mock.ExpectQuery(`FROM quad_links`).
		WithArgs(int64(21), int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "customer_id", "asset_id"}))
	mock.ExpectExec(`UPDATE quad_links SET status = 'LINKED'`).
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectQuery(`INSERT INTO scan_logs`).
		WithArgs(int64(7), int64(2), "张师傅", int64(9), "MATCH").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(100)))
	mock.ExpectCommit()
}

func TestVerifyScan_AssetLinkage(t *testing.T) {
	t.Run("MATCH 联动资产 DEPLOYED(带地址)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		scanHappyMock(mock)

		sink := &fakeSink{}
		s := NewPGStore(mock).UseAssetSink(sink)
		res, err := s.VerifyScan(context.Background(), ScanReq{
			OrderID: 7, WorkerID: 2, WorkerName: "张师傅", ScannedEPC: "EPC-OK",
		})
		if err != nil || res != "MATCH" {
			t.Fatalf("res=%s err=%v", res, err)
		}
		if len(sink.deployed) != 1 || sink.deployed[0] != [2]int64{5, 21} {
			t.Fatalf("deployed=%v, want [(5,21)]", sink.deployed)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("联动失败 → 强一致回滚阻断 MATCH", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`FROM ports`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(11)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), int64(5), int64(3), int64(11), int64(21), int64(1), "主品牌·企业", "UNLINKED"))
		mock.ExpectQuery(`FROM tags`).
			WithArgs("EPC-OK").
			WillReturnRows(mock.NewRows([]string{"id", "bound_asset_id"}).AddRow(int64(9), int64(5)))
		mock.ExpectBegin()
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(21), int64(1)).
			WillReturnRows(mock.NewRows([]string{"id", "customer_id", "asset_id"}))
		mock.ExpectExec(`UPDATE quad_links SET status = 'LINKED'`).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectRollback() // 联动失败,整单回滚(链路/日志一并撤销)

		sink := &fakeSink{deployErr: asset.ErrAssetScrapped}
		s := NewPGStore(mock).UseAssetSink(sink)
		if _, err := s.VerifyScan(context.Background(), ScanReq{
			OrderID: 7, WorkerID: 2, WorkerName: "张师傅", ScannedEPC: "EPC-OK",
		}); err == nil {
			t.Fatalf("want error when sink fails")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("拆机联动资产回 IN_STOCK", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), int64(5), int64(3), int64(11), int64(21), int64(1), "主品牌·企业", "LINKED"))
		mock.ExpectQuery(`FROM tags`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"epc_code"}).AddRow("EPC-OK"))
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE quad_links SET status = 'UNLINKED'`).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		sink := &fakeSink{}
		s := NewPGStore(mock).UseAssetSink(sink)
		if err := s.UnbindRequireScan(context.Background(), 7, "EPC-OK"); err != nil {
			t.Fatalf("UnbindRequireScan: %v", err)
		}
		if len(sink.released) != 1 || sink.released[0] != 5 {
			t.Fatalf("released=%v, want [5]", sink.released)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

var _ = errors.Is // 占位:错误断言经由 errors.Is 场景在上层覆盖
