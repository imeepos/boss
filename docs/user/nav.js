// 用户端 · 底部导航（首页 / 产品 / 订单 / 我的）
(function () {
  var tabs = [
    { key: 'home',     label: '首页', glyph: '⌂', href: 'home.html' },
    { key: 'products', label: '产品', glyph: '▦', href: 'products.html' },
    { key: 'orders',   label: '订单', glyph: '≡', href: 'orders.html' },
    { key: 'profile',  label: '我的', glyph: '◉', href: 'profile.html' },
  ];
  var bar = document.getElementById('tabbar');
  if (!bar) return;
  var cur = location.pathname.split('/').pop().replace('.html', '') || 'home';
  bar.innerHTML = tabs.map(function (t) {
    var cls = t.key === cur ? 'active' : '';
    return '<a class="' + cls + '" href="' + t.href + '">'
      + '<span class="ti">' + t.glyph + '</span>' + t.label + '</a>';
  }).join('');
})();
