package monthly

// 域单测:CSV 契约(BOM/表头逐字节/行级校验)、派生列、导出与模板字节级闭环。

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

// fixtureUserRevenue 与 docs/books 模板同构的最小样本(BOM+LF 由构建函数补)。
func fixtureCSV(rows ...string) []byte {
	var buf bytes.Buffer
	buf.Write(bom)
	buf.WriteString(tableSpecs[TableUserRevenue].header[0])
	for _, h := range tableSpecs[TableUserRevenue].header[1:] {
		buf.WriteString("," + h)
	}
	for _, r := range rows {
		buf.WriteString("\r\n")
		buf.WriteString(r)
	}
	return buf.Bytes()
}

func TestParseCSV_BOMAndRows(t *testing.T) {
	data := fixtureCSV("2026-08,Anilao,518,34,15,0,568645,45492,51000,12283,3071")
	rows, err := parseCSVTable(TableUserRevenue, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0][0] != "2026-08" || rows[0][1] != "Anilao" || rows[0][2] != "518" {
		t.Fatalf("rows=%v", rows)
	}
}

func TestParseCSV_HeaderMismatch(t *testing.T) {
	data := append(append([]byte{}, bom...), []byte("month,region,...\r\n2026-08,Anilao,1,2,3,4,5,6,7,8,9")...)
	if _, err := parseCSVTable(TableUserRevenue, data); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if _, err := parseCSVTable("nope", data); !errors.Is(err, ErrUnknownTable) {
		t.Fatalf("want ErrUnknownTable, got %v", err)
	}
}

func TestDecodeRow_Rejects(t *testing.T) {
	cases := []struct{ name, row string }{
		{"bad month", "2026-13,Anilao,1,2,3,4,5,6,7,8,9"},
		{"negative", "2026-08,Anilao,-1,2,3,4,5,6,7,8,9"},
		{"empty region", "2026-08,,1,2,3,4,5,6,7,8,9"},
		{"short row", "2026-08,Anilao,1,2"},
	}
	for _, c := range cases {
		if _, err := decodeRow(TableUserRevenue, splitRow(c.row)); err == nil {
			t.Fatalf("%s: want error", c.name)
		}
	}
}

// splitRow 测试辅助:按逗号拆(样本无引号字段)。
func splitRow(s string) []string {
	out := []string{}
	cur := []byte{}
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, string(cur))
			cur = cur[:0]
			continue
		}
		cur = append(cur, s[i])
	}
	return append(out, string(cur))
}

func TestDerivedColumns(t *testing.T) {
	if got := ClosingActive(518, 34, 15, 0); got != 537 {
		t.Fatalf("closing=%d", got)
	}
	if got := TotalRevenue(568645, 45492, 51000, 12283, 3071); got != 649783 {
		t.Fatalf("total=%d", got)
	}
	if Ratio(1, 0) != nil {
		t.Fatal("den=0 want nil")
	}
	if v := Ratio(1, 4); v == nil || *v != 0.25 {
		t.Fatalf("ratio=%v", v)
	}
}

// TestExportMatchesTemplateBytes 导入导出字节级闭环:对权威模板原文件
// parse → buildCSV 输出与输入逐字节一致(BOM+CRLF+表头+行序)。
func TestExportMatchesTemplateBytes(t *testing.T) {
	for _, kind := range []string{TableUserRevenue, TableNetworkDelivery, TableFinanceCost} {
		name, err := templateName(kind)
		if err != nil {
			t.Fatal(err)
		}
		orig, err := os.ReadFile("../../../docs/books/" + name)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		rows, err := parseCSVTable(kind, orig)
		if err != nil {
			t.Fatalf("%s parse: %v", kind, err)
		}
		var out []csvRow
		for _, rec := range rows {
			r, err := decodeRow(kind, rec)
			if err != nil {
				t.Fatalf("%s decode: %v", kind, err)
			}
			out = append(out, r)
		}
		got, err := buildCSV(kind, out)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(orig, got) {
			t.Fatalf("%s: roundtrip bytes differ:\norig=% x\ngot =% x", kind, orig, got)
		}
	}
}

func templateName(kind string) (string, error) {
	switch kind {
	case TableUserRevenue:
		return "模板_月度数据_用户与收入.csv", nil
	case TableNetworkDelivery:
		return "模板_月度数据_网络与交付.csv", nil
	case TableFinanceCost:
		return "模板_月度数据_财务与成本.csv", nil
	default:
		return "", ErrUnknownTable
	}
}

func TestValidateMonth(t *testing.T) {
	for _, ok := range []string{"2026-01", "2026-12"} {
		if err := ValidateMonth(ok); err != nil {
			t.Fatalf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"2026-00", "2026-13", "26-08", "2026-8", "2026/08", ""} {
		if err := ValidateMonth(bad); err == nil {
			t.Fatalf("%q want error", bad)
		}
	}
}
