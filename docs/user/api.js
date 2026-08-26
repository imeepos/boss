// 用户端统一接口对接层。
// 契约: api/openapi/user.yaml;假数据: api/mock/combined.js(http://127.0.0.1:8090)。
// 用 window.API 暴露,页面直接调用;后端就绪后仅改 BASE 即可切换到真实网关。
(function (global) {
  'use strict';

  // 生产门户默认使用 102 真实 user API；本地开发可由 API_BASE_URL 覆盖。
  // 不在门户代码内嵌 mock，避免把局部演示误判为真实旅程。
  var DEFAULT_BASE = 'http://192.168.0.102:28080/api/user/v1';
  var BASE = (global.API_BASE_URL || DEFAULT_BASE);
  var TOKEN_KEY = 'boss_user_token';

  function token() {
    return localStorage.getItem(TOKEN_KEY) || '';
  }

  function setToken(t) {
    if (t) localStorage.setItem(TOKEN_KEY, t);
    else localStorage.removeItem(TOKEN_KEY);
  }

  function request(method, path, body) {
    var opt = {
      method: method,
      headers: { 'Content-Type': 'application/json' },
    };
    var tk = token();
    if (tk) opt.headers['Authorization'] = 'Bearer ' + tk;
    if (body !== undefined) opt.body = JSON.stringify(body);
    return fetch(BASE + path, opt).then(function (r) {
      if (!r.ok) throw new Error('HTTP ' + r.status);
      return r.json();
    });
  }

  function get(path) { return request('GET', path); }
  function post(path, body) { return request('POST', path, body || {}); }
  function put(path, body) { return request('PUT', path, body || {}); }

  var API = {
    config: { base: BASE },
    token: token,
    setToken: setToken,

    auth: {
      smsCode: function (phone, scene) { return post('/auth/sms-code', { phone: phone, scene: scene }); },
      login: function (phone, mode, credential) {
        var body = { phone: phone, mode: mode };
        if (mode === 'sms') body.smsCode = credential;
        else body.password = credential;
        return post('/auth/login', body);
      },
      register: function (phone, smsCode, password) { return post('/auth/register', { phone: phone, smsCode: smsCode, password: password }); },
      resetPassword: function (phone, smsCode, newPassword) { return post('/auth/reset-password', { phone: phone, smsCode: smsCode, newPassword: newPassword }); },
      logout: function () { return post('/auth/logout'); },
      verify: function () { return get('/auth/verify'); },
      submitVerify: function (payload) { return post('/auth/verify', payload); },
    },

    profile: {
      get: function () { return get('/profile'); },
      security: function () { return get('/profile/security'); },
      changePassword: function (o, n) { return put('/profile/security/password', { oldPassword: o, newPassword: n }); },
      changePhone: function (p, c) { return put('/profile/security/phone', { newPhone: p, smsCode: c }); },
      notifySettings: function () { return get('/profile/notify-settings'); },
      saveNotifySettings: function (s) { return put('/profile/notify-settings', s); },
      setLanguage: function (lang) { return put('/profile/language', { language: lang }); },
    },

    product: {
      list: function (category) { return get('/products' + (category ? '?category=' + category : '')); },
      detail: function (id) { return get('/products/' + id); },
    },

    addon: {
      list: function () { return get('/addons'); },
      subscribe: function (id) { return post('/addons/' + id + '/subscribe'); },
      unsubscribe: function (id) { return post('/addons/' + id + '/unsubscribe'); },
    },

    order: {
      list: function (status) { return get('/orders' + (status ? '?status=' + status : '')); },
      submit: function (payload) { return post('/orders', payload); },
      detail: function (orderNo) { return get('/orders/' + orderNo); },
      cancel: function (orderNo) { return post('/orders/' + orderNo + '/cancel'); },
      urge: function (orderNo) { return post('/orders/' + orderNo + '/urge'); },
      changeAddress: function (orderNo, addressId) { return post('/orders/' + orderNo + '/change-address', { addressId: addressId }); },
      rate: function (orderNo) { return get('/orders/' + orderNo + '/rate'); },
      submitRate: function (orderNo, payload) { return post('/orders/' + orderNo + '/rate', payload); },
    },

    plan: {
      change: function (planId, payload) { return post('/plans/' + planId + '/change', payload); },
      move: function (planId, payload) { return post('/plans/' + planId + '/move', payload); },
      cancelPreview: function (planId) { return get('/plans/' + planId + '/cancel'); },
      cancel: function (planId, reason) { return post('/plans/' + planId + '/cancel', { reason: reason }); },
    },

    bill: {
      list: function (status) { return get('/bills' + (status ? '?status=' + status : '')); },
      detail: function (billNo) { return get('/bills/' + billNo); },
    },

    payment: {
      create: function (payload) { return post('/payments', payload); },
      list: function () { return get('/payments'); },
      receipt: function (payNo) { return get('/payments/' + payNo + '/receipt'); },
      stripeCheckout: function (payload) { return post('/payments/stripe/checkout', payload); },
    },

    topup: {
      balance: function () { return get('/topups'); },
      pay: function (amount, payMethod) { return post('/topups', { amount: amount, payMethod: payMethod }); },
    },

    invoice: {
      list: function () { return get('/invoices'); },
      apply: function (billNo) { return post('/invoices', { billNo: billNo }); },
    },

    fault: {
      list: function () { return get('/faults'); },
      submit: function (payload) { return post('/faults', payload); },
      detail: function (ticketNo) { return get('/faults/' + ticketNo); },
    },

    complaint: {
      list: function () { return get('/complaints'); },
      submit: function (payload) { return post('/complaints', payload); },
    },

    service: {
      chat: function (message) { return post('/service/chat', { message: message }); },
      faq: function () { return get('/service/faq'); },
    },

    points: {
      get: function () { return get('/points'); },
      tier: function () { return get('/points/tier'); },
      tasks: function () { return get('/points/tasks'); },
      complete: function (id) { return post('/points/tasks/' + encodeURIComponent(id) + '/complete'); },
    },

    misc: {
      home: function () { return get('/home'); },
      messages: function (category) { return get('/messages' + (category ? '?category=' + category : '')); },
      readAllMessages: function () { return post('/messages/read-all'); },
      coupons: function (status) { return get('/coupons' + (status ? '?status=' + status : '')); },
      usage: function (period) { return get('/usage' + (period ? '?period=' + period : '')); },
      addresses: function () { return get('/addresses'); },
      createAddress: function (payload) { return post('/addresses', payload); },
      diySteps: function () { return get('/diy/steps'); },
      agreement: function () { return get('/agreement'); },
    },
  };

  global.API = API;
  return API;
})(typeof window !== 'undefined' ? window : globalThis);
