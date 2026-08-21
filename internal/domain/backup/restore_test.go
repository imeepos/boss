package backup

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCoerceValue(t *testing.T) {
	// bytea:base64 字符串 → []byte
	raw, _ := decodeJSON(t, `"aGVsbG8="`)
	got, err := coerceValue("bytea", raw)
	if err != nil {
		t.Fatalf("bytea decode: %v", err)
	}
	if string(got.([]byte)) != "hello" {
		t.Fatalf("bytea = %v, want hello", got)
	}

	// _text 数组:[]any(全字符串) → []string
	raw, _ = decodeJSON(t, `["a","b"]`)
	arr, err := coerceValue("_text", raw)
	if err != nil {
		t.Fatalf("array: %v", err)
	}
	if len(arr.([]string)) != 2 || arr.([]string)[1] != "b" {
		t.Fatalf("array = %v", arr)
	}

	// 数组元素非字符串:报错
	raw, _ = decodeJSON(t, `["a",1]`)
	if _, err := coerceValue("_text", raw); err == nil {
		t.Fatal("mixed array should error")
	}

	// 其他类型原样透传(nil/数字/字符串)
	raw, _ = decodeJSON(t, `null`)
	if v, _ := coerceValue("int8", raw); v != nil {
		t.Fatalf("nil = %v", v)
	}
	raw, _ = decodeJSON(t, `123`)
	n, _ := coerceValue("int8", raw)
	if n.(json.Number).String() != "123" {
		t.Fatalf("number = %v", n)
	}
}

func decodeJSON(t *testing.T, s string) (any, error) {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
}

func TestCoerceRowMismatch(t *testing.T) {
	if _, err := coerceRow([]string{"int8"}, nil); err == nil {
		t.Fatal("length mismatch should error")
	}
}

func TestInsertSQL(t *testing.T) {
	sql := insertSQL("accounts", []string{"id", "name"})
	want := `INSERT INTO "accounts" ("id", "name") VALUES ($1,$2) ON CONFLICT DO NOTHING`
	if sql != want {
		t.Fatalf("sql = %s", sql)
	}
}
