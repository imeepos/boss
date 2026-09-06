package asset

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ymm-001/boss/internal/pkg/database"
)

// 迁移 000190 配套(P4-T1):文件形态机械断言(CI 必跑)+ 表达式唯一索引真实行为
// (BOSS_PG_TEST_DSN 门控,与 odn/pg_integration_test.go 同模式)。

const macUpFile = "../../../migrations/000190_asset_mac_canonical.up.sql"
const macDownFile = "../../../migrations/000190_asset_mac_canonical.down.sql"

// TestMACMigrationFiles 迁移 SQL 关键形态断言:存量归一 UPDATE 与写入口径一致、
// 旧索引删除、表达式唯一索引(归一后非空)重建、down 完整回滚到列值形态。
func TestMACMigrationFiles(t *testing.T) {
	upSQL, err := os.ReadFile(filepath.Join(macUpFile))
	if err != nil {
		t.Fatalf("read up: %v", err)
	}
	downSQL, err := os.ReadFile(filepath.Join(macDownFile))
	if err != nil {
		t.Fatalf("read down: %v", err)
	}
	up, down := string(upSQL), string(downSQL)
	for _, want := range []string{
		// 存量归一:与应用层 NormalizeMAC 同一口径(去分隔符+大写+两两配冒号)。
		"UPDATE assets",
		"rtrim(regexp_replace(upper(regexp_replace(mac, '[:. -]', '', 'g')), '(..)', '\\1:', 'g'), ':')",
		// 旧列值索引删除。
		"DROP INDEX IF EXISTS uq_assets_mac",
		// 表达式唯一索引:归一空间 = 大写去冒号/横杠/点/空格。
		"CREATE UNIQUE INDEX uq_assets_mac",
		"ON assets (upper(regexp_replace(mac, '[:. -]', '', 'g')))",
		// WHERE 归一后非空(NULL/空串/纯分隔符不占唯一名额)。
		"WHERE upper(regexp_replace(mac, '[:. -]', '', 'g')) <> ''",
	} {
		if !strings.Contains(up, want) {
			t.Errorf("up missing %q", want)
		}
	}
	for _, want := range []string{
		"DROP INDEX IF EXISTS uq_assets_mac",
		"ON assets (mac)",
		"WHERE mac IS NOT NULL AND mac <> ''",
	} {
		if !strings.Contains(down, want) {
			t.Errorf("down missing %q", want)
		}
	}
	if strings.Contains(up, strings.TrimPrefix(macDownFile, "../../../")) ||
		!strings.Contains(up, "BEGIN;") || !strings.Contains(up, "COMMIT;") {
		t.Error("up must be transactional and not reference down")
	}
}

// macSeedFixture 集成造数:借首个法人/批次最小行,返回清理函数。
func macSeedFixture(ctx context.Context, pool *pgxpool.Pool) (int64, func(), error) {
	var ent, batch int64
	if err := pool.QueryRow(ctx, `SELECT min(id) FROM legal_entities`).Scan(&ent); err != nil {
		return 0, nil, err
	}
	err := pool.QueryRow(ctx,
		`INSERT INTO asset_batches(legal_entity_id, code, name) VALUES($1,$2,$3) RETURNING id`,
		ent, "accmac-batch", "MAC 迁移集成测试批次").Scan(&batch)
	if err != nil {
		return 0, nil, err
	}
	cleanup := func() {
		pool.Exec(ctx, `DELETE FROM assets WHERE batch_id=$1`, batch)
		pool.Exec(ctx, `DELETE FROM asset_batches WHERE id=$1`, batch)
	}
	return batch, cleanup, nil
}

// insertMacAsset 直插一行 mac 资产(绕过应用层,专测 DB 索引行为)。
func insertMacAsset(ctx context.Context, pool *pgxpool.Pool, batch int64, code, mac string) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO assets(asset_code, batch_id, legal_entity_id, legal_entity_name, type, status, mac)
		 SELECT $1, $2, b.legal_entity_id, b.name, 'ONU', 'IN_STOCK', $3
		   FROM asset_batches b WHERE b.id = $2`,
		code, batch, mac)
	return err
}

// TestPGStore_MacExpressionUnique_Integration 表达式唯一索引真实行为:
// 同 MAC 不同形态(横杠小写 vs 冒号大写)第二行 23505 且约束名 uq_assets_mac
// (与 classifyAssetInsertErr 的 409 分类键一致);NULL mac 不占名额可多行。
func TestPGStore_MacExpressionUnique_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("database.Migrate: %v", err)
	}
	batch, cleanup, err := macSeedFixture(ctx, pool)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	t.Cleanup(cleanup)
	if err := insertMacAsset(ctx, pool, batch, "accmac-A-1", "aa-bb-cc-11-22-33"); err != nil {
		t.Fatalf("insert dash form: %v", err)
	}
	err = insertMacAsset(ctx, pool, batch, "accmac-A-2", "AA:BB:CC:11:22:33")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "uq_assets_mac" {
		t.Fatalf("same mac dash vs colon: want 23505 uq_assets_mac, got %v", err)
	}
	err = insertMacAsset(ctx, pool, batch, "accmac-A-3", "AABBCC112233")
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("same mac bare hex: want 23505, got %v", err)
	}
	if err := insertMacAsset(ctx, pool, batch, "accmac-A-4", ""); err != nil {
		t.Fatalf("empty mac should not occupy slot: %v", err)
	}
	if err := insertMacAsset(ctx, pool, batch, "accmac-A-5", ""); err != nil {
		t.Fatalf("second empty mac should also pass: %v", err)
	}
}