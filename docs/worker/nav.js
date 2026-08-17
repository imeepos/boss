// 师傅端 · 底部导航（工作台 / 工单 / 我的）+ 多语言切换（示意）
(function () {
  var tabs = [
    { key: 'home',   label: '工作台', glyph: '⌂', href: 'home.html' },
    { key: 'orders', label: '工单',   glyph: '≡', href: 'orders.html' },
    { key: 'profile',label: '我的',   glyph: '◉', href: 'profile.html' },
  ];
  var bar = document.getElementById('tabbar');
  if (!bar) return;
  var cur = location.pathname.split('/').pop().replace('.html', '') || 'home';
  bar.innerHTML = tabs.map(function (t) {
    var cls = t.key === cur ? 'active' : '';
    var badge = t.key === 'orders' ? '<span class="bubble">2</span>' : '';
    return '<a class="' + cls + '" href="' + t.href + '">'
      + '<span class="ti">' + t.glyph + badge + '</span>' + t.label + '</a>';
  }).join('');
})();

// 顶部多语言切换（示意，未完整串接文案包）
window.toggleLang = function () {
  alert('语言切换：中文 / English / Filipino（示意）');
};
