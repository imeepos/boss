// docs/admin 页面静态审计:找出无事件绑定的按钮/控件、缺分页/缺筛选等问题。
// 用法: node scripts/audit-admin-pages.js
const fs = require('fs');
const path = require('path');
const DIR = path.join(__dirname, '..', 'docs', 'admin');

const files = fs.readdirSync(DIR).filter((f) => f.endsWith('.html'));
// 详情页(tab 聚合的小数据集表格,非列表页)豁免分页规则
const NO_PAGER_EXEMPT = new Set(['user-detail.html', 'worker-detail.html']);
const report = [];

for (const f of files) {
  const src = fs.readFileSync(path.join(DIR, f), 'utf8');
  const issues = [];
  // 脚本部分(内联 <script> 无 src 的内容)
  const scripts = [...src.matchAll(/<script(?![^>]*\bsrc=)[^>]*>([\s\S]*?)<\/script>/gi)].map((m) => m[1]).join('\n');
  const used = (id) =>
    scripts.includes(`'${id}'`) || scripts.includes(`"${id}"`) ||
    scripts.includes(`'#${id}'`) || scripts.includes(`"#${id}"`);

  // 1) id 以 btn- 开头但脚本中未通过 id 绑定事件
  const btnIds = [...src.matchAll(/<button[^>]*\bid="(btn-[a-z0-9-]+)"/gi)].map((m) => m[1]);
  for (const id of btnIds) {
    if (!used(id)) issues.push(`按钮 #${id} 无事件绑定`);
  }

  // 2) 表格 tbody 是否有渲染(存在 tbody id 但未在脚本中引用)
  const tbodyIds = [...src.matchAll(/<tbody[^>]*\bid="([a-z0-9-]+)"/gi)].map((m) => m[1]);
  for (const id of tbodyIds) {
    if (!used(id)) issues.push(`tbody #${id} 无数据渲染`);
  }

  // 3) 有表格但无分页容器/无 UI.paginate
  if (!NO_PAGER_EXEMPT.has(f) && /<table class="tbl"/.test(src) && !/UI\.paginate/.test(src)) {
    issues.push('列表页缺少分页 UI.paginate');
  }

  // 4) 有 .pagination 容器但无 paginate 调用
  if (/class="pagination"/.test(src) && !/UI\.paginate/.test(src)) {
    issues.push('有分页容器但未调用 UI.paginate');
  }

  // 5) 工具栏筛选控件(input/select)是否被读取
  const filterIds = [...src.matchAll(/<(?:input|select)[^>]*\bid="([a-z0-9-]+)"/gi)]
    .map((m) => m[1]).filter((id) => !id.startsWith('btn-'));
  for (const id of filterIds) {
    if (!used(id)) issues.push(`筛选控件 #${id} 未被脚本读取`);
  }

  // 6) 操作列链接 data-op 是否有委托处理
  if (/data-op=/.test(src) && !/addEventListener\('click'/.test(src) && !/\.on\('click'/.test(src)) {
    issues.push('存在 data-op 操作链接但无点击委托');
  }

  // 7) 脚本中是否引用 API(假数据页是否对接 mock)
  if (!/API\./.test(src) && f !== 'login.html') {
    issues.push('未调用 API 层(疑似静态页)');
  }

  // 8) onclick 内联残留
  if (/onclick="/.test(src)) issues.push('存在内联 onclick');

  if (issues.length) report.push({ file: f, issues });
}

for (const r of report) {
  console.log(`\n${r.file}`);
  for (const i of r.issues) console.log(`  - ${i}`);
}
console.log(`\n共 ${files.length} 页,${report.length} 页存在静态审计问题`);
