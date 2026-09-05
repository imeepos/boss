package quadlink

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 回归(2026-09-04 任务A):同客户同地址重装不再撞 uq_quad_links_address 裸 23505。
// 102 现场:地址 288 存在历史 LINKED 行(id=379,同客户 213,旧装机 DONE),
// 新订单 scan-bind 置 LINKED 触发唯一键冲突返回 50000。
// 语义裁定:同客户 → 刷新复用既有行;跨客户 → ErrAddressConflict(40920 族)。
// 见 docs/notes/adopted/2026-09-04-quadlink-reinstall-reuse.md。

func TestPGStore_VerifyScan_ReinstallReuse(t *testing.T) {
	t.Run("同客户重装 → 刷新复用既有行并清理本单临时预绑定", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`FROM ports`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
		// 本单 applyTag 落的预绑定行(id=2,UNLINKED)。
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(11)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(2), int64(0), int64(3), int64(11), int64(21), int64(1), "主品牌·企业", "UNLINKED"))
		mock.ExpectQuery(`FROM tags`).
			WithArgs("EPC-OK").
			WillReturnRows(mock.NewRows([]string{"id", "bound_asset_id"}).AddRow(int64(9), int64(5)))
		// 写路径事务化(P1-T1)。
		mock.ExpectBegin()
		// 预绑定无资产(asset_id=0)→ 扫码回填。
		mock.ExpectExec(`UPDATE quad_links SET asset_id`).
			WithArgs(int64(2), int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		// 重装复用守卫:同地址历史活跃行(id=379,同客户 3;第三列为既有行资产快照)。
		mock.ExpectQuery(`ORDER BY id DESC LIMIT 1`).
			WithArgs(int64(21), int64(2)).
			WillReturnRows(mock.NewRows([]string{"id", "customer_id", "asset_id"}).AddRow(int64(379), int64(3), int64(5)))
		// 刷新复用:既有行改绑新端口/资产并置 LINKED。
		mock.ExpectExec(`UPDATE quad_links SET port_id`).
			WithArgs(int64(379), int64(11), int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		// 本单临时预绑定行清理。
		mock.ExpectExec(`DELETE FROM quad_links WHERE id`).
			WithArgs(int64(2)).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectQuery(`INSERT INTO scan_logs`).
			WithArgs(int64(7), int64(2), "张师傅", int64(9), "MATCH").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(100)))
		mock.ExpectCommit()

		s := NewPGStore(mock)
		res, err := s.VerifyScan(context.Background(), ScanReq{
			OrderID: 7, WorkerID: 2, WorkerName: "张师傅", ScannedEPC: "EPC-OK",
		})
		if err != nil {
			t.Fatalf("VerifyScan: %v", err)
		}
		if res != "MATCH" {
			t.Fatalf("res=%s, want MATCH", res)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("跨客户同地址 → ErrAddressConflict(不再裸 23505)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`FROM ports`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(11)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(2), int64(0), int64(3), int64(11), int64(21), int64(1), "主品牌·企业", "UNLINKED"))
		mock.ExpectQuery(`FROM tags`).
			WithArgs("EPC-OK").
			WillReturnRows(mock.NewRows([]string{"id", "bound_asset_id"}).AddRow(int64(9), int64(5)))
		// 写路径事务化(P1-T1)。
		mock.ExpectBegin()
		// 预绑定无资产(asset_id=0)→ 扫码回填。
		mock.ExpectExec(`UPDATE quad_links SET asset_id`).
			WithArgs(int64(2), int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		// 同地址活跃行归属其他客户 999。
		mock.ExpectQuery(`ORDER BY id DESC LIMIT 1`).
			WithArgs(int64(21), int64(2)).
			WillReturnRows(mock.NewRows([]string{"id", "customer_id", "asset_id"}).AddRow(int64(379), int64(999), int64(0)))
		// 冲突在提交前抛出 → 整单回滚(回填一并撤销)。
		mock.ExpectRollback()

		s := NewPGStore(mock)
		_, err := s.VerifyScan(context.Background(), ScanReq{
			OrderID: 7, WorkerID: 2, WorkerName: "张师傅", ScannedEPC: "EPC-OK",
		})
		if !errors.Is(err, ErrAddressConflict) {
			t.Fatalf("err=%v, want ErrAddressConflict", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
