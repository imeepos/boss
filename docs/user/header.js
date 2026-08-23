// 用户端 · 统一顶部导航栏（返回按钮 + 页面标题 + 语言切换）。
// 用法：在页面 <body> 内合适位置放 <div id="header"></div>，然后引入本脚本。
// 会自动从 <title> 提取页面标题，支持 data-i18n 属性。
// 语言切换通过 localStorage 'boss_user_locale' 存储，切换后刷新页面。
(function () {
  'use strict';

  var LOCALES = {
    'zh-CN': { back: '返回', lang: '中文', profile: '我的' },
    'en-US': { back: 'Back', lang: 'English', profile: 'Profile' },
    'ms-MY': { back: 'Kembali', lang: 'Melayu', profile: 'Profil' }
  };

  function getLocale() {
    return localStorage.getItem('boss_user_locale') || 'zh-CN';
  }

  function setLocale(lang) {
    localStorage.setItem('boss_user_locale', lang);
    location.reload();
  }

  function esc(v) {
    return String(v == null ? '' : v).replace(/[&<>"']/g, function (c) {
      return ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c];
    });
  }

  function renderHeader() {
    var el = document.getElementById('header');
    if (!el) return;

    var lang = getLocale();
    var msgs = LOCALES[lang] || LOCALES['zh-CN'];
    var title = document.title.split('·')[0].trim() || document.title;

    // 返回按钮
    var backHref = el.getAttribute('data-back') || 'home.html';
    var backHtml = '<a class="back" href="' + esc(backHref) + '" aria-label="' + esc(msgs.back) + '">‹</a>';

    // 标题
    var titleHtml = '<div class="t">' + esc(title) + '</div>';

    // 语言切换器
    var langHtml = '<div class="lang-switch"><select id="lang-select" aria-label="Language">';
    var keys = Object.keys(LOCALES);
    for (var i = 0; i < keys.length; i++) {
      var k = keys[i];
      langHtml += '<option value="' + k + '"' + (k === lang ? ' selected' : '') + '>' + esc(LOCALES[k].lang) + '</option>';
    }
    langHtml += '</select></div>';

    el.innerHTML = backHtml + titleHtml + langHtml;
    el.className = 'topbar';

    // 绑定语言切换
    var select = document.getElementById('lang-select');
    if (select) {
      select.addEventListener('change', function () {
        setLocale(this.value);
      });
    }
  }

  // DOMContentLoaded 或立即执行
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', renderHeader);
  } else {
    renderHeader();
  }
})();
