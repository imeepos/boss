package adminapi

import (
	"testing"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

func TestSummaryScopeTypes(t *testing.T) {
	_ = aaa.AdminSummary{Accounts: 3, Active: 1, AuthSuccess: 2}
	page := aaa.AdminPage{Page: 1, PageSize: 20}
	if page.Page != 1 {
		t.Fatal("admin page contract changed")
	}
}
