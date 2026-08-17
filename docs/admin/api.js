// 管理后台统一接口对接层。
// 契约: api/openapi/admin.yaml;假数据: api/mock/admin/server.js(http://127.0.0.1:8092)。
// 用 window.API 暴露,页面直接调用;后端就绪后仅改 BASE 即可切换到真实网关。
(function (global) {
  'use strict';

  // 与用户端 api.js 同规则:取页面所在 hostname 直连同机 mock。
  var DEFAULT_BASE = (function () {
    var host = (typeof location !== 'undefined' && location.hostname) || '';
    if (host && host !== 'localhost' && host !== '127.0.0.1') {
      return 'http://' + host + ':8092/api/admin/v1';
    }
    return 'http://127.0.0.1:8092/api/admin/v1';
  })();
  var BASE = (global.ADMIN_API_BASE_URL || DEFAULT_BASE);
  var TOKEN_KEY = 'boss_admin_token';

  function token() { return localStorage.getItem(TOKEN_KEY) || ''; }
  function setToken(t) {
    if (t) localStorage.setItem(TOKEN_KEY, t);
    else localStorage.removeItem(TOKEN_KEY);
  }

  function request(method, path, body) {
    var opt = { method: method, headers: { 'Content-Type': 'application/json' } };
    var tk = token();
    if (tk) opt.headers['Authorization'] = 'Bearer ' + tk;
    if (body !== undefined) opt.body = JSON.stringify(body);
    return fetch(BASE + path, opt).then(function (r) {
      if (!r.ok) throw new Error('HTTP ' + r.status);
      return r.json();
    });
  }

  function get(path, params) {
    if (params) {
      var qs = Object.keys(params)
        .filter(function (k) { return params[k] !== undefined && params[k] !== ''; })
        .map(function (k) { return encodeURIComponent(k) + '=' + encodeURIComponent(params[k]); })
        .join('&');
      if (qs) path += (path.indexOf('?') >= 0 ? '&' : '?') + qs;
    }
    return request('GET', path);
  }
  function post(path, body) { return request('POST', path, body || {}); }
  function put(path, body) { return request('PUT', path, body || {}); }
  function del(path) { return request('DELETE', path); }

  // 域分组方法(与 api/openapi/admin/*.yaml 一一对应)
  var API = {
    config: { base: BASE },
    token: token,
    setToken: setToken,
    get: get,
    post: post,

    auth: {
      login: function (u, p) { return post('/auth/login', { username: u, password: p }); },
      logout: function () { return post('/auth/logout'); },
      me: function () { return get('/auth/me'); },
    },
    dashboard: {
      get: function () { return get('/dashboard'); },
    },
    sys: {
      accounts: function (kw) { return get('/accounts', { keyword: kw }); },
      roles: function () { return get('/roles'); },
      addresses: function () { return get('/addresses'); },
      params: function () { return get('/params'); },
      updateParam: function (k, v) { return put('/params/' + k, { value: v }); },
      auditLogs: function (kw, type) { return get('/audit-logs', { keyword: kw, type: type }); },
      importTasks: function (kw) { return get('/import-tasks', { keyword: kw }); },
      createImportTask: function (o) { return post('/import-tasks', o); },
    },
    org: {
      legalEntities: function (kw) { return get('/legal-entities', { keyword: kw }); },
      departments: function (le) { return get('/departments', { legalEntityId: le }); },
      posts: function (d) { return get('/posts', { deptId: d }); },
      regions: function (kw) { return get('/regions', { keyword: kw }); },
      menuPerms: function () { return get('/menu-perms'); },
      dataScopes: function (kw) { return get('/data-scopes', { keyword: kw }); },
    },
    customer: {
      list: function (kw) { return get('/customers', { keyword: kw }); },
      verifyLogs: function (id) { return get('/customers/' + id + '/verify-logs'); },
      products: function () { return get('/products'); },
      priceHistory: function (id) { return get('/products/' + id + '/price-history'); },
      changePrice: function (id, newPrice) { return post('/products/' + id + '/price-history', { newPrice: newPrice }); },
    },
    billing: {
      bills: function () { return get('/bills'); },
      payments: function (kw) { return get('/payments', { keyword: kw }); },
      arrears: function () { return get('/arrears'); },
      stop: function (id) { return post('/arrears/' + id + '/stop'); },
      resume: function (id) { return post('/arrears/' + id + '/resume'); },
      stopResumeTasks: function (kw, st) { return get('/stop-resume-tasks', { keyword: kw, status: st }); },
      retryStopResume: function (id) { return post('/stop-resume-tasks/' + id + '/retry'); },
      reconciliations: function (kw, st) { return get('/reconciliations', { keyword: kw, status: st }); },
      settle: function (no) { return post('/reconciliations/' + no + '/settle'); },
    },
    asset: {
      list: function (kw, st) { return get('/assets', { keyword: kw, status: st }); },
      lifecycle: function (code) { return get('/assets/' + code + '/lifecycle'); },
      tags: function () { return get('/tags'); },
      stocktakes: function () { return get('/stocktakes'); },
      replacements: function (kw, st) { return get('/replacements', { keyword: kw, status: st }); },
    },
    oss: {
      ports: function () { return get('/ports'); },
      portChangeHistory: function (code) { return get('/ports/' + code + '/change-history'); },
      reserves: function (kw, st) { return get('/reserves', { keyword: kw, status: st }); },
      releaseReserve: function (id) { return post('/reserves/' + id + '/release'); },
      transfers: function (st, type) { return get('/transfers', { status: st, type: type }); },
      approveTransfer: function (no) { return post('/transfers/' + no + '/approve'); },
      oltDevices: function () { return get('/olt-devices'); },
      loAccounts: function (kw, st) { return get('/lo-accounts', { keyword: kw, status: st }); },
      expansions: function (kw, st) { return get('/expansions', { keyword: kw, status: st }); },
    },
    order: {
      list: function (kw, st) { return get('/orders', { keyword: kw, status: st }); },
      detail: function (no) { return get('/orders/' + no); },
      dispatchPool: function () { return get('/dispatch/pool'); },
      myTickets: function () { return get('/dispatch/my-tickets'); },
      dispatchTransfers: function () { return get('/dispatch/transfers'); },
      dismantles: function (kw, st) { return get('/dismantles', { keyword: kw, status: st }); },
      complaints: function (st, type) { return get('/complaints', { status: st, type: type }); },
      closeComplaint: function (no) { return post('/complaints/' + no + '/close'); },
      callbacks: function (kw, st) { return get('/activation-callbacks', { keyword: kw, status: st }); },
      retryCallback: function (id) { return post('/activation-callbacks/' + id + '/retry'); },
    },
    quad: {
      links: function (code) { return get('/quad-links', { code: code }); },
      conflicts: function () { return get('/quad-conflicts'); },
      resolveConflict: function (no) { return post('/quad-conflicts/' + no + '/resolve'); },
      scanLogs: function (kw) { return get('/scan-logs', { keyword: kw }); },
    },
    provision: {
      tasks: function () { return get('/provision-tasks'); },
      templates: function () { return get('/provision-templates'); },
      logs: function (kw, st) { return get('/provision-logs', { keyword: kw, status: st }); },
    },
    alarm: {
      list: function (lv) { return get('/alarms', { level: lv }); },
      ack: function (id) { return post('/alarms/' + id + '/ack'); },
      batchRetest: function (scope) { return post('/alarms/batch-retest', { scope: scope }); },
    },
    aaa: {
      logs: function (kw, type) { return get('/aaa-logs', { keyword: kw, type: type }); },
    },
    intel: {
      gisAddressLinks: function () { return get('/gis/address-links'); },
      analytics: function (region, brand) { return get('/analytics/overview', { region: region, brand: brand }); },
      reports: function (kw) { return get('/reports', { keyword: kw }); },
      sendReport: function (id) { return post('/reports/' + id + '/send'); },
    },
  };

  global.API = API;
})(window);
