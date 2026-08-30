package main

// A2 方法级对账单测:spec 解析(路径→方法归属)与 a2Core 判定(缺失/错位/豁免)。

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanSpecRoutesFile(t *testing.T) {
	dir := t.TempDir()
	yaml := "openapi: 3.0.0\n" +
		"paths:\n" +
		"  /orders:\n" +
		"    get:\n" +
		"      operationId: listOrders\n" +
		"    post:\n" +
		"      operationId: createOrder\n" +
		"  /push/device:\n" +
		"    post:\n" +
		"      operationId: registerDevice\n" +
		"      requestBody:\n" +
		"        content:\n" +
		"          application/json:\n" +
		"            schema:\n" +
		"              type: object\n"
	file := filepath.Join(dir, "user.yaml")
	if err := os.WriteFile(file, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	if err := scanSpecRoutesFile(file, out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"GET /orders", "POST /orders", "POST /push/device"} {
		if !out[want] {
			t.Fatalf("missing %q in %v", want, out)
		}
	}
	// 方法块内的深层子项(schema: type: object)不得误捕成方法。
	if len(out) != 3 {
		t.Fatalf("over-captured: %v", out)
	}
}

// 回归(A5):单引号包裹的 path key(geo.yaml 深层路径形态)必须正确归属方法,
// 否则深层路径的方法块被误记到上一个未引号 path——A2 首扫 geo names 假阳性根因。
func TestScanSpecRoutesFileQuotedKeys(t *testing.T) {
	dir := t.TempDir()
	yaml := "openapi: 3.0.0\n" +
		"paths:\n" +
		"  /geo/countries/{code}/names:\n" +
		"    post:\n" +
		"      operationId: addGeoCountryName\n" +
		"  '/geo/countries/{code}/names/{locale}/{nameType}':\n" +
		"    delete:\n" +
		"      operationId: deleteGeoCountryName\n"
	file := filepath.Join(dir, "admin.yaml")
	if err := os.WriteFile(file, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	if err := scanSpecRoutesFile(file, out); err != nil {
		t.Fatal(err)
	}
	if !out["POST /geo/countries/{code}/names"] {
		t.Fatalf("missing parent POST: %v", out)
	}
	if !out["DELETE /geo/countries/{code}/names/{locale}/{nameType}"] {
		t.Fatalf("missing quoted deep DELETE: %v", out)
	}
	if out["DELETE /geo/countries/{code}/names"] {
		t.Fatalf("deep DELETE misattributed to parent: %v", out)
	}
}

func TestA2Core(t *testing.T) {
	implPaths := map[string]bool{"/orders": true, "/push/device": true}
	implMethods := map[string]bool{"POST /orders": true, "POST /push/device": true}
	specMethods := map[string]bool{
		"GET /orders":       true, // 方法错位:契约 GET,实现仅 POST
		"POST /orders":      true,
		"POST /push/device": true,
		"GET /ghost":        true, // 整路径缺失:契约登记但未实现
	}
	base := map[string]bool{"a2:spec:user GET /ghost": true} // ghost 已豁免
	fails, exempt := a2Core("user", implPaths, implMethods, specMethods, base)
	if fails != 1 || exempt != 1 {
		t.Fatalf("a2Core fails=%d exempt=%d, want 1/1(仅 GET /orders 错位)", fails, exempt)
	}
	// 错位项被 baseline 豁免后应归零。
	base["a2:method:user GET /orders"] = true
	fails, exempt = a2Core("user", implPaths, implMethods, specMethods, base)
	if fails != 0 || exempt != 2 {
		t.Fatalf("豁免后 fails=%d exempt=%d, want 0/2", fails, exempt)
	}
}
