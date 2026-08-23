(function () {
  'use strict';
  function locale() { return localStorage.getItem('boss_worker_locale') || 'zh-CN'; }
  function setLocale(value) {
    localStorage.setItem('boss_worker_locale', value);
    document.dispatchEvent(new CustomEvent('boss:locale-change', { detail: { locale: value } }));
    render();
    if (window.L) window.L.translatePage();
  }
  function render() {
    var el = document.getElementById('header');
    if (!el) return;
    var labels = { 'zh-CN': ['返回', '中文'], 'en-US': ['Back', 'English'], 'ms-MY': ['Kembali', 'Melayu'] };
    var text = labels[locale()] || labels['zh-CN'];
    var back = el.getAttribute('data-back') || 'home.html';
    var titleKey = el.getAttribute('data-i18n');
    var title = window.L && titleKey ? window.L.t(titleKey, document.title.split('·')[0].trim()) : document.title.split('·')[0].trim();
    el.className = 'topbar';
    el.innerHTML = '<a class="back" href="' + back + '">' + text[0] + '</a><div class="t" data-i18n="' + (titleKey || '') + '">' + title + '</div><select id="worker-lang" aria-label="Language"><option value="zh-CN">中文</option><option value="en-US">English</option><option value="ms-MY">Melayu</option></select>';
    document.getElementById('worker-lang').value = locale();
    document.getElementById('worker-lang').addEventListener('change', function () { setLocale(this.value); });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', render); else render();
})();
