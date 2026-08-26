// 用户端 · i18n 引擎(纯逻辑,词条见 locale.<lang>.js 子文件)。
// 用法:
//   <script src="locale.zh-CN.js"></script>
//   <script src="locale.en-US.js"></script>
//   <script src="locale.ms-MY.js"></script>
//   <script src="locale.js"></script>
// 页面引入本文件前需先加载至少一个 locale.<lang>.js(将词条挂到 window.LOCALES[lang]);
// 当前语言由 localStorage 'boss_user_locale' 控制,缺位回落 zh-CN;
// 所有 [data-i18n] 元素在 DOMContentLoaded 后自动翻译。
(function (global) {
  'use strict';

  // LOCALES 由 locale.<lang>.js 子文件归并;未引入子文件时为空对象(容错降级)。
  var LOCALES = global.LOCALES || {};

  function getLocale() {
    return localStorage.getItem('boss_user_locale') || 'zh-CN';
  }

  function setLocale(lang) {
    localStorage.setItem('boss_user_locale', lang);
    location.reload();
  }

  function t(key, fallback) {
    var lang = getLocale();
    var dict = LOCALES[lang] || LOCALES['zh-CN'];
    return dict[key] || LOCALES['zh-CN'][key] || fallback || key;
  }

  function formatDate(value, options) {
    return new Intl.DateTimeFormat(getLocale(), options).format(new Date(value));
  }

  function formatNumber(value, options) {
    return new Intl.NumberFormat(getLocale(), options).format(value);
  }

  function translatePage() {
    document.querySelectorAll('[data-i18n]').forEach(function (el) {
      var key = el.getAttribute('data-i18n');
      var val = t(key, el.textContent || el.getAttribute('placeholder') || '');
      if (val !== key) {
        if (el.tagName === 'INPUT' && el.placeholder) {
          el.placeholder = val;
        } else {
          el.textContent = val;
        }
      }
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', translatePage);
  } else {
    translatePage();
  }

  global.L = {
    t: t,
    locale: getLocale,
    setLocale: setLocale,
    translatePage: translatePage,
    LOCALES: LOCALES
  };
})(window);