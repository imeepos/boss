package aaa

import "testing"

func TestNormalizePage(t *testing.T) {
	got := normalizePage(AdminPage{Page: 0, PageSize: 500, Keyword: " x ", Status: " ACTIVE ", Loid: " L-1 "})
	if got.Page != 1 || got.PageSize != 20 || got.Keyword != "x" || got.Status != "ACTIVE" || got.Loid != "L-1" {
		t.Fatalf("normalized=%+v", got)
	}
}

func TestLoAccountWhereScopes(t *testing.T) {
	where, args := loAccountWhere(AdminPage{Keyword: "L", Status: "ACTIVE"}, AdminScope{LegalEntityID: 2, RegionScope: "root.ncr"})
	if len(args) != 4 || where == "" {
		t.Fatalf("where=%q args=%v", where, args)
	}
	if !contains(where, "legal_entity_id") || !contains(where, "region_path <@") {
		t.Fatalf("missing scope predicates: %s", where)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
