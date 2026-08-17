// 管理后台页面完整性检查:语法 + 功能标记。
// 用法: node scripts/check_admin_pages.js docs/admin/a.html docs/admin/b.html ...
// 退出码 0=全部通过, 1=存在失败项。
const fs = require('fs');

function extractInlineScripts(html) {
  const blocks = [];
  const re = /<script(?![^>]*\bsrc=)[^>]*>([\s\S]*?)<\/script>/g;
  let m;
  while ((m = re.exec(html))) blocks.push(m[1]);
  return blocks.join('\n');
}

function check(file) {
  const fails = [];
  const warns = [];
  const html = fs.readFileSync(file, 'utf8');
  const name = file.split('/').pop();

  // 1) 内联脚本语法(屏蔽浏览器全局后做 Function 语法解析)
  const src = extractInlineScripts(html);
  if (src.trim()) {
    try {
      new Function('window', 'document', 'localStorage', 'location', 'fetch', 'API', 'UI', 'Menu', src.replace(/\bwindow\./g, 'window.'));
    } catch (e) {
      fails.push('JS语法错误: ' + e.message);
    }
  } else {
    fails.push('无内联脚本(页面无逻辑)');
  }

  // 2) 引入共享组件层(列表类页面必须)
  const isListPage = /<table|tbody/.test(html) && !/user-detail|worker-detail|login/.test(name);
  if (isListPage && !/common\.js/.test(html)) fails.push('列表页未引入 common.js');
  if (isListPage && !/api\.js/.test(html)) fails.push('未引入 api.js');

  // 3) 功能标记(按页面类型)
  if (isListPage) {
    if (!/toolbar/.test(html)) fails.push('缺少工具栏 .toolbar');
    if (!/class="input"/.test(html) || !/(kw|keyword|filter|search)/i.test(html)) fails.push('缺少筛选输入框');
    if (!/<select/.test(html) && !/status|st\b/.test(html)) fails.push('缺少状态下拉筛选');
    if (!/pagination|pager/.test(html)) fails.push('缺少分页 .pagination');
    if (!/(modal|UI\.modal|UI\.confirm)/.test(html)) fails.push('缺少弹框(UI.modal/UI.confirm)');
    if (!/(UI\.formFields|<form|form-item)/.test(html)) fails.push('缺少表单(UI.formFields/form-item)');
    if (!/(详情|查看)/.test(html)) fails.push('缺少行操作-详情/查看');
    if (/detail[\s\S]{0,600}UI\.modal/.test(src)) warns.push('详情不应使用弹框,应为下钻页/页内详情面板(返回+从属数据)');
    if (!/tbodyState|无数据|无匹配/.test(html)) fails.push('缺少空态');
    if (!/(toast|alert)/.test(html)) fails.push('缺少错误提示');
    if (!/刷新|refresh/.test(html)) warns.push('建议提供刷新按钮');
    if (!/btn-search|查询|addEventListener\('input'|addEventListener\('change'/.test(html)) fails.push('缺少查询触发');
  }
  // 详情/仪表盘页至少要有数据加载与错误处理
  if (/user-detail|worker-detail/.test(name)) {
    if (!/api\.js/.test(html)) fails.push('未引入 api.js');
    if (!/toast|alert/.test(html)) fails.push('缺少错误提示');
  }
  return { name, fails, warns };
}

const files = process.argv.slice(2);
if (!files.length) { console.error('usage: node check_admin_pages.js <html...>'); process.exit(1); }
let bad = 0;
for (const f of files) {
  const r = check(f);
  if (r.fails.length) {
    bad++;
    console.log('FAIL ' + r.name);
    r.fails.forEach(x => console.log('  - ' + x));
  } else {
    console.log('PASS ' + r.name + (r.warns.length ? ' (' + r.warns.join('; ') + ')' : ''));
  }
}
process.exit(bad ? 1 : 0);
