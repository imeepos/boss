package user

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// 回归:部门/岗位等复合唯一键约束名不含 "username",isUniqueViolation 须按 SQLSTATE 23505 判定,
// 否则批量导入重复行会被映射为 500 而非 409。
func TestIsUniqueViolation_AnyConstraint(t *testing.T) {
	cases := map[string]bool{
		"accounts_username_key":         true,
		"uq_departments_entity_name":    true,
		"uq_posts_dept_code":            true,
		"":                              true,
		"uq_legal_entities_code":        true,
		"uq_customers_phone":            true,
		"uq_product_offers_entity_name": true,
	}
	for name, want := range cases {
		err := &pgconn.PgError{Code: "23505", ConstraintName: name}
		if got := isUniqueViolation(err); got != want {
			t.Fatalf("constraint %q: got %v want %v", name, got, want)
		}
	}
	if isUniqueViolation(&pgconn.PgError{Code: "23503", ConstraintName: "accounts_username_key"}) {
		t.Fatal("23503 must not be treated as unique violation")
	}
	if isUniqueViolation(errors.New("boom")) {
		t.Fatal("plain error must not be treated as unique violation")
	}
}
