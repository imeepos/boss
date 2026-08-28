// openapidoc 指针与引用工具:外部 $ref 拆分、JSON Pointer 寻址、yaml 结构规整。
package openapidoc

import (
	"fmt"
	"strconv"
	"strings"
)

func isExternal(ref string) bool { return !strings.HasPrefix(ref, "#") }

func splitRef(ref string) (file, ptr string) {
	if i := strings.Index(ref, "#"); i >= 0 {
		return ref[:i], ref[i+1:]
	}
	return ref, ""
}

// isRefHolder 判断节点是否为单一 $ref 占位。
func isRefHolder(m map[string]any) bool {
	if len(m) != 1 {
		return false
	}
	_, ok := m[refKey].(string)
	return ok
}

// schemaName 判断指针是否指向 components/schemas/<Name>,是则返回名字。
func schemaName(ptr string) (string, bool) {
	segs := strings.Split(strings.TrimPrefix(ptr, "/"), "/")
	if len(segs) == 3 && segs[0] == "components" && segs[1] == "schemas" {
		return unescape(segs[2]), true
	}
	return "", false
}

// lookup 按 JSON Pointer 取值(~1→/、~0→~)。
func lookup(doc map[string]any, ptr string) (any, error) {
	var cur any = doc
	for _, seg := range strings.Split(ptr, "/") {
		if seg == "" {
			continue
		}
		switch t := cur.(type) {
		case map[string]any:
			v, ok := t[unescape(seg)]
			if !ok {
				return nil, fmt.Errorf("指针段未命中: %s", seg)
			}
			cur = v
		case []any:
			i, err := strconv.Atoi(seg)
			if err != nil || i < 0 || i >= len(t) {
				return nil, fmt.Errorf("指针段非法: %s", seg)
			}
			cur = t[i]
		default:
			return nil, fmt.Errorf("指针穿越标量: %s", seg)
		}
	}
	return cur, nil
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, "~1", "/")
	return strings.ReplaceAll(s, "~0", "~")
}

// normalize 兜底转换 yaml 二阶映射(v3 已产 map[string]any,此为防御)。
func normalize(v any) any {
	switch t := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[fmt.Sprint(k)] = normalize(val)
		}
		return out
	case map[string]any:
		for k, val := range t {
			t[k] = normalize(val)
		}
		return t
	case []any:
		for i := range t {
			t[i] = normalize(t[i])
		}
		return t
	default:
		return v
	}
}
