#!/usr/bin/env bash
# docs/user 演示门户冒烟:逐页加载 HTML,跑页面内联 <script>,断言无 JS 解析错。
# 用法: scripts/check-docs-user.sh
# 环境:Node >=22(系统 Node 可;不依赖 102,不调真实 API,纯静态语法冒烟)。
# 不在 102 上跑——演示门户是 docs/ 静态产物,本机即可。
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
USER_DIR="$ROOT/docs/user"
cd "$USER_DIR" || exit 1

FAIL=0
PAGES=$(ls *.html 2>/dev/null | wc -l)
echo "docs/user: $PAGES pages"

# 1) locale.js / 子词条文件必须无 JS 解析错。
for f in locale.js locale.zh-CN.js locale.en-US.js locale.ms-MY.js api.js; do
  if ! node -c "$f" 2>/dev/null; then
    echo "  FAIL: $f parse error"; FAIL=$((FAIL+1))
  else
    echo "  OK:   $f"
  fi
done

# 2) 每个 HTML 页:<script src="*.js"> 必须全部解析通过。
# 用 vm 模块构造 fake window/document/localStorage,串行加载所有 src= 的脚本,
# 任一抛错即该页 FAIL。
check_page() {
  local page="$1"
  python3 - "$page" <<'PY' 2>&1
import re, sys, subprocess, os, tempfile
page = sys.argv[1]
with open(page, 'r') as f:
    html = f.read()
scripts = re.findall(r'<script[^>]*src="([^"]+)"', html)
# 找出 inline scripts 同步段(<script>...</script> 不带 src)
inline_blocks = re.findall(r'<script(?![^>]*\bsrc=)[^>]*>(.*?)</script>', html, re.S)
errors = []
for src in scripts:
    path = os.path.join('.', src)
    if not os.path.exists(path):
        errors.append(f"missing script: {src}")
        continue
    try:
        out = subprocess.run(['node', '-c', path], capture_output=True, text=True, timeout=10)
        if out.returncode != 0:
            errors.append(f"{src} parse error: {out.stderr.strip()[:120]}")
    except Exception as e:
        errors.append(f"{src} check exception: {e}")
for i, blk in enumerate(inline_blocks):
    blk = blk.strip()
    if not blk:
        continue
    with tempfile.NamedTemporaryFile('w', suffix='.js', delete=False) as t:
        # 用 fake globalThis 防 'window is not defined'
        t.write("var window=globalThis, document={addEventListener:()=>{},readyState:'complete',querySelectorAll:()=>[]}, localStorage={getItem:()=>null,setItem:()=>{}};\n")
        t.write(blk)
        path = t.name
    try:
        out = subprocess.run(['node', '-c', path], capture_output=True, text=True, timeout=10)
        if out.returncode != 0:
            errors.append(f"inline script #{i} parse error: {out.stderr.strip()[:120]}")
    finally:
        os.unlink(path)
if errors:
    print(f"FAIL {page}:")
    for e in errors: print(f"  - {e}")
    sys.exit(1)
print(f"OK   {page}")
PY
}

for page in *.html; do
  if ! check_page "$page" >/tmp/_page_check.$$ 2>&1; then
    cat /tmp/_page_check.$$; FAIL=$((FAIL+1))
  else
    cat /tmp/_page_check.$$
  fi
done
rm -f /tmp/_page_check.$$

if [ "$FAIL" -gt 0 ]; then
  echo "TOTAL FAIL: $FAIL"; exit 1
fi
echo "TOTAL OK: $PAGES pages"