package geo

// 纯单测:limit 钳制、列表 SQL 拼装与默认国家值解析(不依赖 PG;SQL 行为由 DSN 集成测试复核)。

import (
	"strings"
	"testing"
)

func TestClampSubdivLimit(t *testing.T) {
	cases := map[int]int{-5: 50, 0: 50, 1: 1, 7: 7, 50: 50, 200: 200, 201: 200, 5000: 200}
	for in, want := range cases {
		if got := clampSubdivLimit(in); got != want {
			t.Fatalf("clamp(%d)=%d want %d", in, got, want)
		}
	}
}

func TestBuildSubdivisionListSQL(t *testing.T) {
	t.Run("兼容口径 country+locale+hasChildren+limit", func(t *testing.T) {
		sql, args := buildSubdivisionListSQL(SubdivisionFilter{CountryCode: "PH", Locale: "zh-Hans"}, 50)
		for _, frag := range []string{
			"n.locale = $1", "d.country_code = $2", "AS has_children", "ORDER BY d.code", "LIMIT $3",
		} {
			if !strings.Contains(sql, frag) {
				t.Fatalf("sql missing %q: %s", frag, sql)
			}
		}
		if len(args) != 3 || args[0] != "zh-Hans" || args[1] != "PH" || args[2] != 50 {
			t.Fatalf("args=%v", args)
		}
	})
	t.Run("parentCode 空串取顶层 非空取直接子节点", func(t *testing.T) {
		empty := ""
		sql, args := buildSubdivisionListSQL(SubdivisionFilter{CountryCode: "ZZ", ParentCode: &empty}, 50)
		if !strings.Contains(sql, "d.parent_code IS NULL") || strings.Contains(sql, "d.parent_code = $") {
			t.Fatalf("top-level sql=%s", sql)
		}
		code := "ZZ-Q0"
		sql, args = buildSubdivisionListSQL(SubdivisionFilter{CountryCode: "ZZ", ParentCode: &code}, 50)
		if !strings.Contains(sql, "d.parent_code = $2") || args[1] != "ZZ-Q0" {
			t.Fatalf("drilldown sql=%s args=%v", sql, args)
		}
	})
	t.Run("nil ParentCode 不过滤 兼容旧行为", func(t *testing.T) {
		sql, _ := buildSubdivisionListSQL(SubdivisionFilter{CountryCode: "PH"}, 50)
		if strings.Contains(sql, "d.parent_code IS NULL") || strings.Contains(sql, "d.parent_code = $") {
			t.Fatalf("unexpected parent filter: %s", sql)
		}
	})
	t.Run("keyword 编码译名双通道且占位符复用", func(t *testing.T) {
		sql, args := buildSubdivisionListSQL(SubdivisionFilter{CountryCode: "PH", Keyword: "quez"}, 50)
		if strings.Count(sql, "ILIKE $2") != 2 {
			t.Fatalf("keyword placeholder not reused: %s", sql)
		}
		if len(args) != 3 || args[1] != "%quez%" {
			t.Fatalf("args=%v", args)
		}
	})
	t.Run("keyword 通配符转义", func(t *testing.T) {
		_, args := buildSubdivisionListSQL(SubdivisionFilter{CountryCode: "PH", Keyword: "a%b_c"}, 50)
		bs := string(rune(92))
		want := "%a" + bs + "%b" + bs + "_c%"
		if args[1] != want {
			t.Fatalf("escaped=%v want %v", args[1], want)
		}
	})
}

func TestParseDefaultCountry(t *testing.T) {
	cases := []struct {
		raw  string
		want DefaultCountry
	}{
		{"", DefaultCountry{}},
		{"PH", DefaultCountry{CountryCode: "PH", Configured: true}},
		{`"PH"`, DefaultCountry{CountryCode: "PH", Configured: true}},
		{`"ph"`, DefaultCountry{CountryCode: "PH", Configured: true}},
		{`" ph "`, DefaultCountry{CountryCode: "PH", Configured: true}},
		{`"USA"`, DefaultCountry{}},
		{`"P1"`, DefaultCountry{}},
		{`[1,2]`, DefaultCountry{}},
		{`""`, DefaultCountry{}},
	}
	for _, c := range cases {
		if got := parseDefaultCountry(c.raw); got != c.want {
			t.Fatalf("parse(%q)=%+v want %+v", c.raw, got, c.want)
		}
	}
}
