// 师傅端统一接口对接层(web/worker 工程版)。
// 契约: api/openapi/worker.yaml。默认走同源 /api/worker/v1:
// dev/preview 由 vite 代理到 mock(BOSS_API_TARGET 覆盖),生产由网关同源转发;
// 跨域部署时可在加载前设置 window.API_BASE_URL 覆盖。
(function (global) {
  'use strict';

  var BASE = (global.API_BASE_URL || '/api/worker/v1');
  var TOKEN_KEY = 'boss_worker_token';
  var PUBLIC_PAGES = { 'login.html': 1 };

  function token() { return localStorage.getItem(TOKEN_KEY) || ''; }
  function setToken(t) {
    if (t) localStorage.setItem(TOKEN_KEY, t);
    else localStorage.removeItem(TOKEN_KEY);
  }

  function currentPage() {
    return (location.pathname.split('/').pop() || 'home.html').toLowerCase();
  }

  // 登录守卫:非公开页无 token 一律回登录页(登录成功后再跳回)
  function guard() {
    if (!PUBLIC_PAGES[currentPage()] && !token()) {
      location.replace('login.html');
      return true;
    }
    return false;
  }

  function request(method, path, body) {
    var opt = { method: method, headers: { 'Content-Type': 'application/json' } };
    var tk = token();
    if (tk) opt.headers['Authorization'] = 'Bearer ' + tk;
    if (body !== undefined) opt.body = JSON.stringify(body);
    return fetch(BASE + path, opt).then(function (r) {
      if (r.status === 401) {
        setToken('');
        if (!PUBLIC_PAGES[currentPage()]) location.replace('login.html');
        throw new Error('登录已失效,请重新登录');
      }
      if (!r.ok) throw new Error('HTTP ' + r.status);
      return r.json();
    });
  }

  function get(path) { return request('GET', path); }
  function post(path, body) { return request('POST', path, body || {}); }
  function put(path, body) { return request('PUT', path, body || {}); }

  // 工具:URL query 拼接(跳过空值)
  function qs(params) {
    var s = '';
    Object.keys(params || {}).forEach(function (k) {
      if (params[k] !== undefined && params[k] !== null && params[k] !== '') {
        s += (s ? '&' : '?') + k + '=' + encodeURIComponent(params[k]);
      }
    });
    return s;
  }

  var API = {
    config: { base: BASE },
    token: token,
    setToken: setToken,

    auth: {
      smsCode: function (phone) { return post('/auth/sms-code', { phone: phone }); },
      login: function (phone, mode, credential) {
        var body = { phone: phone, mode: mode };
        if (mode === 'sms') body.smsCode = credential;
        else body.password = credential;
        return post('/auth/login', body);
      },
      logout: function () { return post('/auth/logout'); },
    },

    home: {
      get: function () { return get('/home'); },
    },

    ticket: {
      list: function (status) { return get('/tickets' + qs({ status: status })); },
      history: function (period) { return get('/tickets/history' + qs({ period: period })); },
      detail: function (no) { return get('/tickets/' + no); },
      accept: function (no) { return post('/tickets/' + no + '/accept'); },
      checkin: function (no, lat, lng) { return post('/tickets/' + no + '/checkin', { lat: lat, lng: lng }); },
      navi: function (no) { return get('/tickets/' + no + '/navi'); },
      transfer: function (no, reason, targetWorkerId, remark) { return post('/tickets/' + no + '/transfer', { reason: reason, targetWorkerId: targetWorkerId || null, remark: remark }); },
      reschedule: function (no, newDate, newSlot, reason, remark) { return post('/tickets/' + no + '/reschedule', { newDate: newDate, newSlot: newSlot, reason: reason, remark: remark }); },
      rollback: function (no) { return post('/tickets/' + no + '/rollback'); },
      retry: function (no) { return post('/tickets/' + no + '/retry'); },
      complaint: function (no, category, content) { return post('/tickets/' + no + '/complaint', { category: category, content: content }); },
      repairReport: function (no, result, remark) { return post('/tickets/' + no + '/repair-report', { result: result, remark: remark }); },
    },

    hall: {
      list: function () { return get('/hall'); },
      grab: function (no) { return post('/hall/' + no + '/grab'); },
    },

    scan: {
      bind: function (no, epc, offline) { return post('/tickets/' + no + '/scan-bind', { epc: epc, offline: !!offline }); },
      abnormal: function (no, payload) { return post('/tickets/' + no + '/scan-abnormal', payload); },
      photos: function (no) { return get('/tickets/' + no + '/photos'); },
      uploadPhoto: function (no, scene) { return post('/tickets/' + no + '/photos', { scene: scene }); },
      report: function (no) { return get('/tickets/' + no + '/report'); },
      submitReport: function (no, remark) { return post('/tickets/' + no + '/report', { remark: remark }); },
      activation: function (no) { return get('/tickets/' + no + '/activation'); },
      activate: function (no) { return post('/tickets/' + no + '/activate'); },
      sign: function (no, signatureData) { return post('/tickets/' + no + '/sign', { signatureData: signatureData }); },
      charge: function (no) { return get('/tickets/' + no + '/charge'); },
      submitCharge: function (no, amount, payMethod) { return post('/tickets/' + no + '/charge', { amount: amount, payMethod: payMethod }); },
    },

    asset: {
      dismantleScan: function (no, epc) { return post('/tickets/' + no + '/dismantle/scan', { epc: epc }); },
      replace: function (no) { return get('/tickets/' + no + '/replace'); },
      submitReplace: function (no, oldEpc, newEpc) { return post('/tickets/' + no + '/replace', { oldEpc: oldEpc, newEpc: newEpc }); },
      returnAsset: function (epc) { return post('/assets/' + epc + '/return'); },
      materials: function () { return get('/materials'); },
      materialOut: function (id) { return post('/materials/' + id + '/out'); },
      tools: function () { return get('/materials/tools'); },
      borrowTool: function (id) { return post('/materials/tools/' + id + '/borrow'); },
      giveBackTool: function (id) { return post('/materials/tools/' + id + '/give-back'); },
      maintenance: function () { return get('/maintenance'); },
      measure: function (no) { return get('/tickets/' + no + '/measure'); },
      resources: function (no) { return get('/tickets/' + no + '/resources'); },
    },

    profile: {
      get: function () { return get('/profile'); },
      performance: function (period) { return get('/performance' + qs({ period: period })); },
      schedule: function (month) { return get('/schedule' + qs({ month: month })); },
      clock: function (type) { return post('/schedule/clock', { type: type }); },
      settings: function () { return get('/settings'); },
      saveSettings: function (s) { return put('/settings', s); },
      feedbacks: function () { return get('/feedbacks'); },
    },

    misc: {
      messages: function () { return get('/messages'); },
      readAll: function () { return post('/messages/read-all'); },
      clear: function () { return post('/messages/clear'); },
      notices: function () { return get('/notices'); },
      faq: function (keyword) { return get('/help/faq' + qs({ keyword: keyword })); },
      serviceMessages: function () { return get('/service/messages'); },
      sendServiceMessage: function (content) { return post('/service/messages', { content: content }); },
      safetyCheck: function (workType, checklist) { return post('/safety/checks', { workType: workType, checklist: checklist }); },
    },
  };

  global.API = API;
  guard();
})(window);
