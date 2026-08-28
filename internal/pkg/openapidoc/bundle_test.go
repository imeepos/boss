package openapidoc

// 聚合器契约测试:以真实内嵌契约树为源,锁定三条不变式——
// ① 外部 $ref 全部解析(残余 ./ 引用即悬空);② 组件并根(本地块+上提去重);
// ③ 内部 #/ 引用保留(合并后自然可解析)。

import (
	"strings"
	"testing"

	"github.com/ymm-001/boss/api/openapi"
)

func bundleFor(t *testing.T, portal string) map[string]any {
	t.Helper()
	doc, err := New(openapi.FS).Doc(portal)
	if err != nil {
		t.Fatalf("bundle %s: %v", portal, err)
	}
	return doc
}

// walkRefs 收集文档树中全部 $ref 值。
func walkRefs(v any, out *[]string) {
	switch t := v.(type) {
	case map[string]any:
		if ref, ok := t[refKey].(string); ok {
			*out = append(*out, ref)
		}
		for _, val := range t {
			walkRefs(val, out)
		}
	case []any:
		for _, val := range t {
			walkRefs(val, out)
		}
	}
}

func TestBundleAdminDoc(t *testing.T) {
	doc := bundleFor(t, "admin.yaml")
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) < 100 {
		t.Fatalf("paths 数量异常: %d", len(paths))
	}

	var refs []string
	walkRefs(doc, &refs)
	for _, ref := range refs {
		if strings.HasPrefix(ref, "./") {
			t.Fatalf("残余外部引用: %s", ref)
		}
	}
	if len(refs) == 0 {
		t.Fatal("内部 #/ 引用不应为空(契约以组件复用为常态)")
	}

	comps, _ := doc["components"].(map[string]any)
	schemas, _ := comps["schemas"].(map[string]any)
	if _, ok := schemas["LicenseStatus"]; !ok {
		t.Fatal("本地 components 块未并根:缺 LicenseStatus")
	}
	if len(schemas) < 10 {
		t.Fatalf("上提+本地 schemas 数量异常: %d", len(schemas))
	}
	if _, ok := comps["securitySchemes"]; !ok {
		t.Fatal("根 securitySchemes 丢失")
	}
}

func TestBundleAllPortals(t *testing.T) {
	for _, portal := range []string{"user.yaml", "worker.yaml", "open.yaml"} {
		doc := bundleFor(t, portal)
		paths, _ := doc["paths"].(map[string]any)
		if len(paths) == 0 {
			t.Fatalf("%s paths 为空", portal)
		}
		var refs []string
		walkRefs(doc, &refs)
		for _, ref := range refs {
			if strings.HasPrefix(ref, "./") {
				t.Fatalf("%s 残余外部引用: %s", portal, ref)
			}
		}
	}
}

func TestBundleUnknownPortal(t *testing.T) {
	b := New(openapi.FS)
	if _, err := b.Doc("nope.yaml"); err == nil {
		t.Fatal("未知聚合根应报错")
	}
}
