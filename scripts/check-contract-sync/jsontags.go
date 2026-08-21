// 检查 B:json tag 必须 lowerCamelCase(fields.md §0 全局强制)。
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var tagNameRe = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func checkJSONTags(root string) int {
	fails := 0
	for _, dir := range []string{"internal", "pkg", "cmd"} {
		filepath.Walk(filepath.Join(root, dir), func(p string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			fails += tagsInFile(p)
			return nil
		})
	}
	if fails == 0 {
		fmt.Println("B OK json tag 全部 lowerCamelCase")
	}
	return fails
}

func tagsInFile(path string) int {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		fmt.Println("B: 解析失败", path, err)
		return 1
	}
	fails := 0
	ast.Inspect(f, func(n ast.Node) bool {
		st, ok := n.(*ast.StructType)
		if !ok {
			return true
		}
		for _, fld := range st.Fields.List {
			if fld.Tag == nil {
				continue
			}
			name := tagName(unquote(fld.Tag.Value))
			if name == "" || name == "-" {
				continue
			}
			if !tagNameRe.MatchString(name) {
				pos := fset.Position(fld.Tag.Pos())
				fmt.Printf("B FAIL %s:%d json:\"%s\" 非 lowerCamelCase\n", path, pos.Line, name)
				fails++
			}
		}
		return true
	})
	return fails
}

func tagName(tag string) string {
	for _, part := range strings.Fields(tag) {
		if v, ok := strings.CutPrefix(part, "json:"); ok {
			return strings.Split(unquote(v), ",")[0]
		}
	}
	return ""
}
