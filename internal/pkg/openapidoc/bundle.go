// Package openapidoc 将仓库内多文件 OpenAPI 3 契约聚合为自包含文档树:
// 解析聚合根的 ./file.yaml#Pointer 转发;域内 ./schemas.yaml#/components/schemas/X
// 上提到根 components(同内容去重);各文件自带 components 块统一并入根;
// 内部 #/ 引用原样保留(合并后自然可解析)。
package openapidoc

import (
	"fmt"
	"io/fs"
	"log"
	"path"
	"reflect"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

const refKey = "$ref"

// Bundler 全端聚合结果缓存:首请求构建一次,后续零解析开销。
// 结果树只读共享,调用方不得修改。
type Bundler struct {
	fsys fs.FS
	once sync.Once
	docs map[string]map[string]any
	err  error
}

// New 以 fsys(内嵌契约树)为源构建 Bundler。
func New(fsys fs.FS) *Bundler { return &Bundler{fsys: fsys} }

// Doc 返回 portal(聚合根文件名,如 admin.yaml)的自包含文档树。
func (b *Bundler) Doc(portal string) (map[string]any, error) {
	b.once.Do(func() { b.docs, b.err = b.buildAll() })
	if b.err != nil {
		return nil, b.err
	}
	doc, ok := b.docs[portal]
	if !ok {
		return nil, fmt.Errorf("openapidoc: 未知聚合根 %s", portal)
	}
	return doc, nil
}

func (b *Bundler) buildAll() (map[string]map[string]any, error) {
	roots, err := fs.Glob(b.fsys, "*.yaml")
	if err != nil {
		return nil, err
	}
	docs := make(map[string]map[string]any, len(roots))
	for _, root := range roots {
		doc, err := b.bundle(root)
		if err != nil {
			return nil, fmt.Errorf("聚合 %s: %w", root, err)
		}
		docs[root] = doc
	}
	return docs, nil
}

// bundler 单端聚合状态:comps 汇集 hoist 与各文件本地 components。
type bundler struct {
	fsys  fs.FS
	files map[string]map[string]any
	comps map[string]map[string]any
	stack []string
}

func (b *Bundler) bundle(rootFile string) (map[string]any, error) {
	w := &bundler{fsys: b.fsys, files: map[string]map[string]any{}, comps: map[string]map[string]any{}}
	doc, err := w.load(rootFile)
	if err != nil {
		return nil, err
	}
	out, err := w.walk(normalize(doc), path.Dir(rootFile))
	if err != nil {
		return nil, err
	}
	root, ok := out.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: 文档根不是映射", rootFile)
	}
	if err := w.collectLocalComponents(rootFile); err != nil {
		return nil, err
	}
	w.mergeComponents(root)
	return root, nil
}

// load 解析并缓存单个契约文件。
func (w *bundler) load(file string) (map[string]any, error) {
	if m, ok := w.files[file]; ok {
		return m, nil
	}
	src, err := fs.ReadFile(w.fsys, file)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	w.files[file] = doc
	return doc, nil
}

// walk 深拷贝式遍历:外部 $ref 原位展开;components/schemas 指向上提为内部引用。
func (w *bundler) walk(v any, dir string) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		if ref, ok := t[refKey].(string); ok && len(t) == 1 && isExternal(ref) {
			return w.resolve(ref, dir)
		}
		out := make(map[string]any, len(t))
		for k, val := range t {
			nv, err := w.walk(val, dir)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			nv, err := w.walk(val, dir)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	default:
		return v, nil
	}
}

// resolve 跟进外部引用:目标为 components/schemas/<Name> 时上提去重,
// 其余(如 paths 项)原位内联;成环立即报错。
func (w *bundler) resolve(ref, dir string) (any, error) {
	file, ptr := splitRef(path.Join(dir, ref))
	key := file + ptr
	for _, k := range w.stack {
		if k == key {
			return nil, fmt.Errorf("$ref 成环: %s", strings.Join(append(w.stack, key), " -> "))
		}
	}
	w.stack = append(w.stack, key)
	defer func() { w.stack = w.stack[:len(w.stack)-1] }()

	doc, err := w.load(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ref, err)
	}
	target, err := lookup(doc, ptr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ref, err)
	}
	if name, ok := schemaName(ptr); ok {
		if err := w.hoist(name, target, path.Dir(file)); err != nil {
			return nil, err
		}
		return map[string]any{refKey: "#/components/schemas/" + name}, nil
	}
	return w.walk(target, path.Dir(file))
}

// hoist 上提被引用 schema:同内容复用;同名不同义记冲突日志并保留先见者(契约层 bug 信号)。
func (w *bundler) hoist(name string, raw any, dir string) error {
	bundled, err := w.walk(raw, dir)
	if err != nil {
		return err
	}
	if prev, ok := w.comps["schemas"][name]; ok {
		if !reflect.DeepEqual(prev, bundled) {
			log.Printf("[openapidoc] CONFLICT schema %s 同名不同义,保留先见者", name)
		}
		return nil
	}
	if w.comps["schemas"] == nil {
		w.comps["schemas"] = map[string]any{}
	}
	w.comps["schemas"][name] = bundled
	return nil
}

// collectLocalComponents 把端内全部文件的本地 components 块登记入 comps
// (路径未 $ref 的域内 schema,如 license.yaml 的 LicenseStatus,漏收=悬空引用)。
// 整块为 $ref 占位(如根文件 schemas: {$ref: './schemas.yaml#...'})时跳过:
// 其内容由根遍历的 resolve 内联,这里再登记会把 "$ref" 当成组件名写脏。
func (w *bundler) collectLocalComponents(rootFile string) error {
	dir := strings.TrimSuffix(rootFile, ".yaml")
	files, err := fs.Glob(w.fsys, path.Join(dir, "*.yaml"))
	if err != nil {
		return err
	}
	files = append(files, rootFile)
	for _, file := range files {
		doc, err := w.load(file)
		if err != nil {
			return err
		}
		comps, _ := doc["components"].(map[string]any)
		for kind, items := range comps {
			m, ok := items.(map[string]any)
			if !ok || isRefHolder(m) {
				continue
			}
			for name, val := range m {
				if name == refKey {
					continue
				}
				w.register(kind, name, val, path.Dir(file))
			}
		}
	}
	return nil
}

// register 本地组件登记(与 hoist 同池,先见者优先;冲突留日志)。
func (w *bundler) register(kind, name string, raw any, dir string) {
	if _, ok := w.comps[kind][name]; ok {
		return
	}
	bundled, err := w.walk(raw, dir)
	if err != nil {
		log.Printf("[openapidoc] FAILED 聚合本地组件 %s/%s: %v", kind, name, err)
		return
	}
	if w.comps[kind] == nil {
		w.comps[kind] = map[string]any{}
	}
	w.comps[kind][name] = bundled
}

// mergeComponents comps 并入根文档 components(缺类建类,同名先见者胜)。
func (w *bundler) mergeComponents(root map[string]any) {
	comps, _ := root["components"].(map[string]any)
	if comps == nil {
		comps = map[string]any{}
		root["components"] = comps
	}
	for kind, items := range w.comps {
		dst, _ := comps[kind].(map[string]any)
		if dst == nil {
			dst = map[string]any{}
			comps[kind] = dst
		}
		for name, val := range items {
			if _, ok := dst[name]; !ok {
				dst[name] = val
			}
		}
	}
}
