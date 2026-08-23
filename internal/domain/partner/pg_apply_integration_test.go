package partner

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPartnerApprovalIntegration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	suffix := strconv.FormatInt(time.Now().UnixNano()%1000000000000, 10)
	credit := "E2E" + suffix
	company := "E2E Partner " + suffix
	var appID, reviewerID, entityID, adminID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM accounts WHERE role_id=(SELECT id FROM roles WHERE code='sysadmin') LIMIT 1`).Scan(&reviewerID); err != nil {
		t.Fatal(err)
	}
	err = pool.QueryRow(ctx, `INSERT INTO partner_applications(company_name, credit_code, contact_name, contact_phone, business_desc) VALUES($1,$2,$3,$4,$5) RETURNING id`, company, credit, "E2E Partner", "+639170"+suffix[len(suffix)-6:], "E2E onboarding").Scan(&appID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM accounts WHERE id=$1`, adminID)
		_, _ = pool.Exec(ctx, `DELETE FROM partner_applications WHERE id=$1`, appID)
		_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE id=$1`, entityID)
	}()
	res, err := NewPGStore(pool).Approve(ctx, appID, reviewerID)
	if err != nil {
		t.Fatal(err)
	}
	entityID, adminID = res.LegalEntityID, res.AdminAccountID
	if res.Username == "" || len(res.InitialPassword) != 12 {
		t.Fatalf("approve result=%+v", res)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM partner_applications WHERE id=$1`, appID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != StatusApproved {
		t.Fatalf("status=%s", status)
	}
	var entityCount, accountCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM legal_entities WHERE id=$1`, entityID).Scan(&entityCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM accounts WHERE id=$1 AND legal_entity_id=$2`, adminID, entityID).Scan(&accountCount); err != nil {
		t.Fatal(err)
	}
	if entityCount != 1 || accountCount != 1 {
		t.Fatalf("entity=%d account=%d", entityCount, accountCount)
	}
}
