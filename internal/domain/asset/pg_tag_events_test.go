package asset

// P1-T2 事件流与回收闭环单测:解绑/报废的状态+事件原子性,幂等与冲突路径。
// pgxmock 正则为子串匹配,锚点一律用无特殊字符的短片段(避 $/() 转义)。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_UnbindTag(t *testing.T) {
	t.Run("已绑定 → 解绑+UNBIND 事件", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`bound_asset_id, 0. FROM tags WHERE id`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"bound_asset_id"}).AddRow(int64(5)))
		mock.ExpectExec(`UPDATE tags SET bound_asset_id = NULL`).
			WithArgs(int64(1), int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO tag_events`).
			WithArgs(int64(1), int64(5), nil, "换新回收", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		s := NewPGStore(mock)
		if err := s.UnbindTag(context.Background(), 1, 0, 0, "换新回收"); err != nil {
			t.Fatalf("UnbindTag: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("未绑定 → ErrTagUnbound", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`bound_asset_id, 0. FROM tags WHERE id`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"bound_asset_id"}).AddRow(int64(0)))
		mock.ExpectRollback()

		s := NewPGStore(mock)
		if err := s.UnbindTag(context.Background(), 1, 0, 0, ""); !errors.Is(err, ErrTagUnbound) {
			t.Fatalf("err=%v, want ErrTagUnbound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("预期资产不符 → ErrBindingConflict", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`bound_asset_id, 0. FROM tags WHERE id`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"bound_asset_id"}).AddRow(int64(5)))
		mock.ExpectRollback()

		s := NewPGStore(mock)
		if err := s.UnbindTag(context.Background(), 1, 9, 0, ""); !errors.Is(err, ErrBindingConflict) {
			t.Fatalf("err=%v, want ErrBindingConflict", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

// scrapLockRows 造报废锁行:五列 = status/asset_code/sn/tag_id/tag_no(P3-F 三要素取数)。
func scrapLockRows(mock pgxmock.PgxPoolIface, status, code, sn string, tagID int64, tagNo *string) *pgxmock.Rows {
	return mock.NewRows([]string{"status", "asset_code", "sn", "tag_id", "tag_no"}).
		AddRow(status, code, sn, tagID, tagNo)
}

// scrapStrP 测试用字符串指针(锁行 tag_no 可空列)。
func scrapStrP(s string) *string { return &s }

func TestPGStore_ScrapAsset(t *testing.T) {
	const lockQ = "FROM assets a LEFT JOIN tags t ON t.id" // 无特殊字符锚点(避 $/() 转义)
	tagged := func(mock pgxmock.PgxPoolIface) *pgxmock.Rows {
		return scrapLockRows(mock, "IN_STOCK", "A-ACC-1", "SN-9", 9, scrapStrP("T-9"))
	}
	bare := func(mock pgxmock.PgxPoolIface) *pgxmock.Rows {
		return scrapLockRows(mock, "IN_STOCK", "A-ACC-1", "", 0, nil)
	}
	t.Run("IN_STOCK 带标签+三要素正确 → SCRAPPED+轨迹+标签 RECYCLE", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(lockQ).WithArgs(int64(7)).WillReturnRows(tagged(mock))
		mock.ExpectExec(`UPDATE assets SET status = 'SCRAPPED'`).
			WithArgs(int64(7)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO asset_lifecycles`).
			WithArgs(int64(7)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE tags SET bound_asset_id = NULL`).
			WithArgs(int64(9), int64(7)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO tag_events`).
			WithArgs(int64(9), int64(7), int64(3), "屏裂报废", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		s := NewPGStore(mock)
		confirm := ScrapConfirm{AssetCode: "A-ACC-1", SN: "SN-9", TagNo: "T-9"}
		if err := s.ScrapAsset(context.Background(), 7, 3, "屏裂报废", confirm); err != nil {
			t.Fatalf("ScrapAsset: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("已 SCRAPPED → 幂等短路,错要素重放同样 no-op", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(lockQ).WithArgs(int64(7)).
			WillReturnRows(scrapLockRows(mock, "SCRAPPED", "A-ACC-1", "SN-9", 0, nil))
		mock.ExpectRollback()

		s := NewPGStore(mock)
		wrong := ScrapConfirm{AssetCode: "A-WRONG", SN: "SN-WRONG", TagNo: "T-WRONG"}
		if err := s.ScrapAsset(context.Background(), 7, 3, "", wrong); err != nil {
			t.Fatalf("ScrapAsset replay: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("三要素不符 → ErrScrapConfirmMismatch 且零写路径", func(t *testing.T) {
		cases := []struct {
			name    string
			rows    func(pgxmock.PgxPoolIface) *pgxmock.Rows
			confirm ScrapConfirm
			msgHit  string
		}{
			{"编码不符", tagged, ScrapConfirm{AssetCode: "A-X", SN: "SN-9", TagNo: "T-9"}, "confirmAssetCode"},
			{"SN 该填不填", tagged, ScrapConfirm{AssetCode: "A-ACC-1", SN: "", TagNo: "T-9"}, "confirmSn 必填"},
			{"SN 不符", tagged, ScrapConfirm{AssetCode: "A-ACC-1", SN: "SN-X", TagNo: "T-9"}, "confirmSn 与资产现值不一致"},
			{"SN 该空不空", bare, ScrapConfirm{AssetCode: "A-ACC-1", SN: "SN-9", TagNo: ""}, "confirmSn 须为空串"},
			{"标签号该填不填", tagged, ScrapConfirm{AssetCode: "A-ACC-1", SN: "SN-9", TagNo: ""}, "confirmTagNo 必填"},
			{"标签号不符", tagged, ScrapConfirm{AssetCode: "A-ACC-1", SN: "SN-9", TagNo: "T-X"}, "confirmTagNo 与标签现值不一致"},
			{"标签号该空不空", bare, ScrapConfirm{AssetCode: "A-ACC-1", SN: "", TagNo: "T-9"}, "confirmTagNo 须为空串"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				mock, _ := pgxmock.NewPool()
				defer mock.Close()
				mock.ExpectBegin()
				mock.ExpectQuery(lockQ).WithArgs(int64(7)).WillReturnRows(tc.rows(mock))
				mock.ExpectRollback() // 不符即回滚,零写路径
				s := NewPGStore(mock)
				err := s.ScrapAsset(context.Background(), 7, 3, "r", tc.confirm)
				if !errors.Is(err, ErrScrapConfirmMismatch) {
					t.Fatalf("err=%v, want ErrScrapConfirmMismatch", err)
				}
				if !strings.Contains(err.Error(), tc.msgHit) {
					t.Fatalf("msg=%q, want hit %q", err.Error(), tc.msgHit)
				}
				if strings.Contains(err.Error(), "SN-9") || strings.Contains(err.Error(), "T-9") {
					t.Fatalf("msg=%q 泄露服务端现值", err.Error())
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatalf("unmet: %v", err)
				}
			})
		}
	})
}
