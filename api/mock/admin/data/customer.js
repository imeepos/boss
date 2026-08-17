// 假数据 —— 客户档案/实名/产品资费(契约: api/openapi/admin/customer.yaml)。
'use strict';

const ok = { code: 0, message: 'success' };

const customers = [
  { customerId: 1, name: '王先生', phone: '138****1234', idType: '身份证', idNo: '110***********1234', realNameStatus: '已实名', serviceStatus: '在网' },
  { customerId: 2, name: '吴女士', phone: '139****5678', idType: '身份证', idNo: '110***********5678', realNameStatus: '已实名', serviceStatus: '欠费' },
  { customerId: 3, name: '孙先生', phone: '137****9012', idType: '—', idNo: '—', realNameStatus: '待补登', serviceStatus: '在网' },
];

const verifyLogs = [
  { customerId: 1, customerName: '王先生', method: '证件 + 人像比对', verifiedAt: '2025-08-01 10:12', result: '通过', operator: 'ops01' },
  { customerId: 2, customerName: '吴女士', method: '证件 OCR', verifiedAt: '2025-07-28 15:40', result: '通过', operator: 'ops01' },
  { customerId: 3, customerName: '孙先生', method: '—', verifiedAt: '—', result: '未核验', operator: '—' },
];

const products = [
  { productId: 1, name: '家庭宽带 500M', bandwidth: '500M', monthlyFee: '¥99', effectiveAt: '2025-08-01', status: '在售' },
  { productId: 2, name: '家庭宽带 1000M', bandwidth: '1000M', monthlyFee: '¥199', effectiveAt: '2025-08-01', status: '在售' },
  { productId: 3, name: '政企专线 100M', bandwidth: '100M', monthlyFee: '¥500', effectiveAt: '2025-09-01', status: '待发布' },
];

const priceHistory = [
  { productId: 1, productName: '家庭宽带 500M', oldPrice: '¥89', newPrice: '¥99', effectiveAt: '2025-08-01 00:00', operator: 'ops01', status: '已生效' },
  { productId: 2, productName: '家庭宽带 1000M', oldPrice: '¥189', newPrice: '¥199', effectiveAt: '2025-08-01 00:00', operator: 'ops01', status: '已生效' },
  { productId: 3, productName: '政企专线 100M', oldPrice: '¥480', newPrice: '¥500', effectiveAt: '2025-09-01 00:00', operator: 'ops02', status: '待生效' },
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
    items: priceHistory.filter((r) => String(r.productId) === String(params.productId)),
  }),

  'POST /products/{productId}/price-history': ok,
};
