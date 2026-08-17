// 用户端路由 —— 按 api/openapi/user.yaml 的方法/路径返回假数据。
// 输入 pathname 已去 /api/v1 前缀;返回 JSON 值或 null(404)。
'use strict';

const data = require('../data.js');

const ok = () => ({ code: 0, message: 'success' });

function route(method, pathname) {
  const strip = (s) => s.replace(/^\/+|\/+$/g, '');
  const p = strip(pathname);

  if (method === 'GET') {
    if (p === 'home') return data.home;
    if (p === 'profile') return data.profile;
    if (p === 'profile/security') return data.security;
    if (p === 'profile/notify-settings') return data.notifySettings;
    if (p === 'auth/verify') return data.verify;
    if (p === 'products') return data.products;
    if (p === 'addons') return data.addons;
    if (p === 'orders') return data.orders;
    if (p === 'bills') return data.bills;
    if (p === 'payments') return data.payments;
    if (p === 'topups') return data.balance;
    if (p === 'invoices') return data.invoice;
    if (p === 'faults') return data.faults;
    if (p === 'complaints') return data.complaints;
    if (p === 'service/faq') return data.faq;
    if (p === 'messages') return data.messages;
    if (p === 'coupons') return data.coupons;
    if (p === 'usage') return data.usage;
    if (p === 'diy/steps') return data.diySteps;
    if (p === 'agreement') return data.agreement;
    if (p === 'addresses') return { items: data.profile.addresses };

    if (p.startsWith('products/')) return data.productDetail;
    if (p.startsWith('orders/')) {
      const seg = p.split('/');
      if (seg[2] === 'rate') {
        const o = data.orders.items[2];
        return { orderNo: o.orderNo, productName: o.productName, address: o.address, finishedAt: '08-11' };
      }
      return data.orderDetail;
    }
    if (p.startsWith('bills/')) return data.billDetail;
    if (p.startsWith('payments/')) {
      const payNo = p.split('/')[1].replace(/\/receipt$/, '');
      return data.receipts[payNo] || data.receipts['PAY20250817001'];
    }
    if (p.startsWith('faults/')) return data.faultDetail;
    if (p.startsWith('plans/')) {
      if (p.endsWith('/cancel')) {
        return {
          unpaidBills: [data.bills.items[0]],
          penalty: 716.4,
          penaltyDesc: '剩余 12 个月 · ¥199 × 30%',
        };
      }
      return data.productDetail;
    }
  }

  if (method === 'POST') {
    if (p === 'auth/sms-code') return ok();
    if (p === 'auth/login') return { token: 'mock-jwt-token-' + Date.now(), customerId: 1 };
    if (p === 'auth/register') return { token: 'mock-jwt-token-' + Date.now(), customerId: 1 };
    if (p === 'auth/reset-password') return ok();
    if (p === 'auth/logout') return ok();
    if (p === 'auth/verify') return ok();
    if (p === 'orders') {
      return {
        orderNo: 'ORD-20250817-0' + (Math.floor(Math.random() * 90) + 10),
        status: 'PENDING',
        statusLabel: '待核查',
        productName: '1000M 极速宽带',
        address: '望京X · 3栋 · 501',
        stage: 1,
        stageLabel: '用户下单',
        canRate: false,
      };
    }
    if (p === 'payments') {
      return { payNo: 'PAY20250817' + String(Math.floor(Math.random() * 900) + 100), amount: 158, billPeriod: '2025-08', payMethod: '微信支付', status: 'SUCCESS' };
    }
    if (p === 'topups') {
      return { payNo: 'PAY20250817' + String(Math.floor(Math.random() * 900) + 100), amount: 100, billPeriod: null, payMethod: '微信支付', status: 'SUCCESS' };
    }
    if (p === 'invoices') return ok();
    if (p === 'faults') {
      return { ticketNo: 'TKT-20250817-0' + (Math.floor(Math.random() * 90) + 10), faultType: 'no_internet', faultTypeLabel: '单户断网（紧急 SLA）', address: '望京X · 3栋 · 501', createdAt: new Date().toISOString().slice(0, 16).replace('T', ' '), status: 'PROCESSING', statusLabel: '处理中' };
    }
    if (p === 'complaints') return ok();
    if (p === 'service/chat') {
      return { reply: '已为您检测到当前服务在线正常。建议先自助排障：检查光猫指示灯，重启光猫与路由器。是否需要转报修？', toHuman: false };
    }
    if (p === 'messages/read-all') return ok();
    if (p === 'addresses') return ok();

    if (p.startsWith('orders/')) {
      const seg = p.split('/');
      if (seg[2] === 'cancel' || seg[2] === 'urge') return ok();
      if (seg[2] === 'change-address') return ok();
      if (seg[2] === 'rate') return ok();
    }
    if (p.startsWith('addons/')) return ok();
    if (p.startsWith('plans/')) {
      return {
        orderNo: 'ORD-20250817-0' + (Math.floor(Math.random() * 90) + 10),
        status: 'PENDING',
        statusLabel: '待核查',
        productName: '1000M 极速宽带',
        address: '望京X · 3栋 · 501',
        stage: 1,
        stageLabel: '用户下单',
        canRate: false,
      };
    }
  }

  if (method === 'PUT') {
    if (p === 'profile/security/password') return ok();
    if (p === 'profile/security/phone') return ok();
    if (p === 'profile/notify-settings') return ok();
    if (p === 'profile/language') return ok();
  }

  return null;
}

module.exports = { route };
