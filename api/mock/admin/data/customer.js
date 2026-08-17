// 假数据 —— 客户档案/实名/产品资费(契约: api/openapi/admin/customer.yaml)。
// 实体由 db.js 派生;调价历史为流程样例。
'use strict';

const db = require('../../db.js');

const ok = { code: 0, message: 'success' };

// 客户: db.customers 全量(订单/工单/欠费引用的客户全部在册)
const customers = db.customers.map((c) => ({
  customerId: c.customerId, name: c.name, phone: c.phoneMasked, idType: c.idType, idNo: c.idNoMasked,
  realNameStatus: c.realNameLabel, serviceStatus: c.serviceLabel,
}));

// 实名日志: 由客户 verifyAt 派生
const verifyLogs = db.customers.map((c) => ({
  customerId: c.customerId, customerName: c.name,
  method: c.verifyAt ? (c.customerId === 2 ? '证件 OCR' : '证件 + 人像比对') : '—',
  verifiedAt: c.verifyAt || '—', result: c.verifyAt ? '通过' : '未核验', operator: c.verifyAt ? 'ops01' : '—',
}));

// 产品: db.products(状态 PUBLISHED→在售 / DRAFT→待发布)
const products = db.products.map((p) => ({
  productId: p.productId, name: p.name, bandwidth: p.bandwidth, monthlyFee: '¥' + p.monthlyFee,
  effectiveAt: '2025-08-01', status: p.status === 'PUBLISHED' ? '在售' : '待发布',
}));

// 调价历史样例(金额与 db.products 现价对齐)
const priceHistory = [
  { productId: 'P-500', productName: '500M 畅享宽带', oldPrice: '¥119', newPrice: '¥129', effectiveAt: '2025-08-01 00:00', operator: 'ops01', status: '已生效' },
  { productId: 'P-1000', productName: '1000M 极速宽带', oldPrice: '¥189', newPrice: '¥199', effectiveAt: '2025-08-01 00:00', operator: 'ops01', status: '已生效' },
  { productId: 'P-BIZ-100', productName: '政企专线 100M', oldPrice: '¥480', newPrice: '¥500', effectiveAt: '2025-09-01 00:00', operator: 'ops02', status: '待生效' },
];

function byKeyword(rows, kw, fields) {
  if (!kw) return rows;
  return rows.filter((r) => fields.some((f) => String(r[f] || '').indexOf(kw) >= 0));
}

module.exports = {
  'GET /customers': ({ query }) => ({ items: byKeyword(customers, query.keyword, ['name', 'phone']) }),

  'GET /customers/{customerId}/verify-logs': ({ params }) => ({
    items: verifyLogs.filter((r) => String(r.customerId) === String(params.customerId)),
  }),

  'GET /products': () => ({ items: products }),

  'POST /products': ok,

  'GET /products/{productId}/price-history': ({ params }) => ({
    items: priceHistory.filter((r) => r.productId === params.productId),
  }),

  'POST /products/{productId}/price-history': ok,
};
