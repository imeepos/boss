// BOSS 管理后台 · 共享菜单（两级折叠，按角色职责 + 能力域重组）
(function () {
  // 分组：{ id, label, ico, items: [{key, label, href}] }
  var groups = [
    { id: 'overview', label: '运营总览', ico: 'icons/overview.svg', items: [
      { key: 'dashboard', label: '工作台', href: 'dashboard.html' },
    ]},
    { id: 'base', label: '基础配置', ico: 'icons/base.svg', items: [
      { key: 'account',  label: '账号与角色', href: 'account.html' },
      { key: 'address',  label: '地址层级',   href: 'address.html' },
      { key: 'params',   label: '业务参数',   href: 'settings.html' },
      { key: 'audit',    label: '审计日志',   href: 'audit.html' },
      { key: 'importer', label: '数据导入中心', href: 'importer.html' },
    ]},
    { id: 'org', label: '组织与权限', ico: 'icons/org.svg', items: [
      { key: 'company',    label: '子公司/法人', href: 'company.html' },
      { key: 'department', label: '部门管理',     href: 'department.html' },
      { key: 'post',       label: '岗位管理',     href: 'post.html' },
      { key: 'region',     label: '经营区域',     href: 'region.html' },
      { key: 'menuperm',   label: '菜单权限',     href: 'menuperm.html' },
      { key: 'datascope',  label: '数据权限',     href: 'datascope.html' },
    ]},
    { id: 'bss', label: '客户与资费', ico: 'icons/bss.svg', items: [
      { key: 'customer', label: '客户档案', href: 'customer.html' },
      { key: 'product',  label: '产品资费', href: 'product.html' },
      { key: 'user',     label: '用户列表', href: 'user.html' },
      { key: 'userdata', label: '用户端配置', href: 'userdata.html' },
    ]},
    { id: 'billing', label: '计费与账务', ico: 'icons/billing.svg', items: [
      { key: 'billing',  label: '出账管理',   href: 'billing.html' },
      { key: 'payment',  label: '缴费管理',   href: 'payment.html' },
      { key: 'arrears',  label: '欠费停复机', href: 'arrears.html' },
      { key: 'stopsrv',  label: '停复机执行', href: 'stopsrv.html' },
      { key: 'paycheck', label: '渠道对账',   href: 'paycheck.html' },
    ]},
    { id: 'ams', label: '资产与标签', ico: 'icons/ams.svg', items: [
      { key: 'asset',    label: '资产台账', href: 'asset.html' },
      { key: 'tag',      label: '电子标签', href: 'tag.html' },
      { key: 'stock',    label: '盘点管理', href: 'stock.html' },
      { key: 'replace',  label: '设备更换单', href: 'replace.html' },
    ]},
    { id: 'oss', label: '网络资源', ico: 'icons/oss.svg', items: [
      { key: 'resource', label: '端口台账', href: 'resource.html' },
      { key: 'reserve',  label: '预占与释放', href: 'reserve.html' },
      { key: 'transfer', label: '跨区域调配', href: 'transfer.html' },
      { key: 'device',   label: 'OLT 设备', href: 'device.html' },
      { key: 'loaccount',label: '认证账号', href: 'loaccount.html' },
      { key: 'expand',   label: '扩容申请', href: 'expand.html' },
    ]},
    { id: 'boss', label: '订单与工单', ico: 'icons/boss.svg', items: [
      { key: 'order',    label: '订单管理', href: 'order.html' },
      { key: 'worker',   label: '师傅管理', href: 'worker.html' },
      { key: 'worker-ops', label: '师傅端内容', href: 'worker-ops.html' },
      { key: 'dispatch', label: '派单管理', href: 'dispatch.html' },
      { key: 'dismantle',label: '拆机管理', href: 'dismantle.html' },
      { key: 'complaint',label: '报障与投诉', href: 'complaint.html' },
      { key: 'callback', label: '激活回调', href: 'callback.html' },
    ]},
    { id: 'quad', label: '四码合一', ico: 'icons/quad.svg', items: [
      { key: 'quadlink', label: '关联查询', href: 'quadlink.html' },
      { key: 'check',    label: '对账与告警', href: 'check.html' },
      { key: 'scanlog',  label: '扫码绑定记录', href: 'scanlog.html' },
    ]},
    { id: 'provision', label: '配置下发', ico: 'icons/provision.svg', items: [
      { key: 'provision',label: '下发任务', href: 'provision.html' },
      { key: 'template', label: '配置模板', href: 'template.html' },
      { key: 'provlog',  label: '下发日志', href: 'provlog.html' },
    ]},
    { id: 'alarm', label: '告警中心', ico: 'icons/alarm.svg', items: [
      { key: 'alarm',    label: '告警列表', href: 'alarm.html' },
    ]},
    { id: 'aaa', label: '认证计费', ico: 'icons/aaa.svg', items: [
      { key: 'aaalog',   label: '话单与认证日志', href: 'aaalog.html' },
    ]},
    { id: 'intel', label: '数字孪生与经营', ico: 'icons/intel.svg', items: [
      { key: 'gis',      label: 'GIS 地图', href: 'gis.html' },
      { key: 'analytics',label: '经营分析', href: 'analytics.html' },
      { key: 'report',   label: '报告中心', href: 'report.html' },
    ]},
  ];

  var container = document.getElementById('menu');
  if (!container) return;

  var path = location.pathname.split('/').pop() || 'dashboard.html';
  var cur = path.replace('.html', '');

  var html = '';
  groups.forEach(function (g) {
    var hasActive = g.items.some(function (it) { return it.href === path; });
    // 含当前页的分组默认展开
    var open = hasActive ? ' open' : '';
    var groupCls = 'mgroup' + open;
    html += '<div class="' + groupCls + '">';
    html += '<div class="mg-title" onclick="this.parentNode.classList.toggle(\'open\')">'
          + (g.ico ? '<span class="ico"><img src="' + g.ico + '" alt=""></span>' : '') + g.label
          + '<span class="arrow">⌄</span></div>';
    html += '<div class="mg-items">';
    g.items.forEach(function (it) {
      var cls = it.href === path ? 'menu-item active' : 'menu-item';
      html += '<a class="' + cls + '" href="' + it.href + '">'
            + '<span class="ico"><img src="icons/items/' + it.key + '.svg" alt=""></span>'
            + it.label + '</a>';
    });
    html += '</div></div>';
  });
  container.innerHTML = html;
})();
