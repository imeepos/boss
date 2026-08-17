// entities.js —— 用户域实体表(单一事实源,由 db.js 聚合导出)
// 用户端用到的全部专属数据在此建表: 账户/地址簿/套餐订购/增值服务/通知订阅/
// FAQ/消息/优惠券/流量/自助排障/协议/余额/发票/投诉/实名记录/账单明细项。
// admin 后经 api/mock/admin/data/userdata.js 管理这些表,用户端视图一律派生,不得手抄。
'use strict';

// —— 用户账户(userAccounts): 登录账户维度,主档见 db.customers ——
const userAccounts = [
  { customerId: 1, username: '138****1234', registeredAt: '2023-05-11', passwordUpdatedAt: '2025-01-03', autoPayEnabled: false, status: 'ACTIVE', statusLabel: '正常' },
  { customerId: 2, username: '139****5678', registeredAt: '2025-07-20', passwordUpdatedAt: '2025-07-20', autoPayEnabled: false, status: 'ACTIVE', statusLabel: '正常' },
];

// —— 地址簿(userAddresses): addrCode 指向安装地址体系(fields.md §4) ——
const userAddresses = [
  { addressId: 'ADDR-001', customerId: 1, addrCode: 'A-3-501', community: '望京X', building: '3栋', door: '501', contact: '王先生', phoneMasked: '138****1234', isDefault: true },
  { addressId: 'ADDR-002', customerId: 1, addrCode: 'A-1-101', community: '望京Y', building: '1栋', door: '101', contact: '王先生', phoneMasked: '138****1234', isDefault: false },
];

// —— 套餐订购(userPlans): productId 外键 db.products ——
const userPlans = [
  { planId: 'PLAN-1000', customerId: 1, productId: 'P-1000', monthlyFee: 199, contractEnd: '2026-08', status: 'ACTIVE', statusLabel: '生效中' },
];

// —— 增值服务目录(addonCatalog)与订购(addonSubscriptions) ——
const addonCatalog = [
  { addonId: 'A-IPTV', name: 'IPTV 高清电视', monthlyFee: 10, description: '¥10/月 · 200+ 频道', active: true },
  { addonId: 'A-WIFI', name: '全屋 WiFi', monthlyFee: 15, description: '¥15/月 · 信号覆盖', active: true },
  { addonId: 'A-CLOUD', name: '云盘存储', monthlyFee: 8, description: '¥8/月 · 500GB', active: true },
  { addonId: 'A-CAM', name: '家庭安全看护', monthlyFee: 20, description: '¥20/月 · 摄像头云存储', active: true },
];
const addonSubscriptions = [
  { subId: 'AS-001', customerId: 1, addonId: 'A-IPTV', subscribedAt: '2023-05-11', status: 'ACTIVE' },
  { subId: 'AS-002', customerId: 1, addonId: 'A-WIFI', subscribedAt: '2025-02-01', status: 'ACTIVE' },
];

// —— 通知订阅(notifyPrefs): 业务/营销/渠道三组开关 ——
const notifyPrefs = [
  { customerId: 1, business: { bill: true, suspendResume: true, faultNotice: true, installProgress: true }, marketing: { promo: true, planRecommend: false }, channels: { inApp: true, sms: true, push: true } },
];

// —— 用户 FAQ(userFaqs): admin 维护,active 控制用户端可见 ——
const userFaqs = [
  { faqId: 'faq-1', question: '如何修改套餐？', answer: '进入「我的套餐」→「改套餐」选择新套餐并提交。', sort: 1, active: true },
  { faqId: 'faq-2', question: '如何开具电子发票？', answer: '进入「电子发票」选择可开票账期申请开票。', sort: 2, active: true },
  { faqId: 'faq-3', question: '账单怎么看？', answer: '进入「我的账单」查看账单明细与费用构成。', sort: 3, active: true },
  { faqId: 'faq-4', question: '如何迁址移机？', answer: '进入「我的套餐」→「迁址」，填写新地址提交。', sort: 4, active: true },
  { faqId: 'faq-5', question: '上网故障如何自检？', answer: '进入「故障报修」→「自助排障」按引导排查。', sort: 5, active: true },
];

// —— 用户消息(userMessages): balance/billing/fault/promo 分类 ——
const userMessages = [
  { messageId: 'M-001', customerId: 1, category: 'balance', title: '余额预警', content: '账户余额 ¥5.00 低于 ¥50，请及时充值避免停机', tag: '预警', tagLevel: 'balance', createdAt: '2025-08-17', read: false },
  { messageId: 'M-002', customerId: 1, category: 'balance', title: '到期停机提醒', content: '账户余额不足，将于明日到期停机，充值后自动复机', tag: '停机', tagLevel: 'balance', createdAt: '2025-08-17', read: false },
  { messageId: 'M-003', customerId: 1, category: 'billing', title: '8 月账单已出账', content: '应缴 ¥158.00 · 账期 08-01 ~ 08-31', tag: '未缴', tagLevel: 'bill', createdAt: '2025-08-17', read: false },
  { messageId: 'M-004', customerId: 1, category: 'billing', title: '装维进度更新', content: '订单 ORD-20250817-001 已进入「扫码绑定」环节', tag: '新', tagLevel: 'info', createdAt: '2025-08-17', read: false },
  { messageId: 'M-005', customerId: 1, category: 'fault', title: '故障公告', content: '望京X 片区 8-17 02:00~04:00 计划割接，可能短暂中断', tag: '公告', tagLevel: 'fault', createdAt: '2025-08-17', read: true },
  { messageId: 'M-006', customerId: 1, category: 'promo', title: '夏日宽带优惠', content: '1000M 套餐首月立减 ¥40，点击领取', tag: '活动', tagLevel: 'promo', createdAt: '2025-08-17', read: true },
];

// —— 优惠券(coupons)与邀请配置(inviteConfig) ——
const coupons = [
  { couponId: 'CPN-001', customerId: 1, amount: 40, threshold: 100, title: '1000M 套餐首月立减', expireAt: '2025-08-31', status: 'available', statusLabel: '可用', active: true },
  { couponId: 'CPN-002', customerId: 1, amount: 20, threshold: 200, title: '余额充值满 200 减 20', expireAt: '2025-09-30', status: 'available', statusLabel: '可用', active: true },
];
const inviteConfig = [{ configId: 'INV-1', link: 'https://boss.example.com/invite/8f3a', rewardDesc: '好友下单双方各得 ¥20 余额' }];

// —— 流量使用(usageRecords): 月度口径,down+up=used ——
const usageRecords = [
  { customerId: 1, period: '2025-08', used: 286.4, unit: 'GB', quota: 1000, dailyAvg: 9.2, down: 248.1, up: 38.3, iptvNote: '不计入流量', onlineDuration: '3 天 14 小时', lastOnlineAt: '2025-08-17 07:41' },
];

// —— 自助排障指南(diyGuides): admin 维护,active 控制可见 ——
const diyGuides = [
  { guideId: 's1', title: '无法上网', desc: '光猫灯异常 / 完全断网', sort: 1, active: true, steps: [
    { title: '1 检查光猫指示灯', desc: 'LOS 红灯说明光纤未接通，需报修' },
    { title: '2 重启光猫与路由器', desc: '断电 30 秒后重新上电' },
    { title: '3 检查认证状态', desc: '确认账号未欠费停机' }] },
  { guideId: 's2', title: '网速慢', desc: '视频卡顿 / 下载缓慢', sort: 2, active: true, steps: [
    { title: '1 检查 WiFi 信号', desc: '靠近路由器重测' },
    { title: '2 减少并发大流量', desc: '暂停下载/大流量应用' }] },
  { guideId: 's3', title: 'IPTV 无信号', desc: '电视黑屏 / 无频道', sort: 3, active: true, steps: [
    { title: '1 检查机顶盒连接', desc: '确认网线与 HDMI 连接' },
    { title: '2 重启机顶盒', desc: '断电重启' }] },
  { guideId: 's4', title: '光猫告警', desc: '红灯 / 闪烁异常', sort: 4, active: true, steps: [
    { title: '1 检查光纤连接', desc: '确认光纤接头插紧' },
    { title: '2 联系报修', desc: 'LOS 红灯持续需报修' }] },
];

// —— 协议(agreements): userAgreement/privacyPolicy,版本化 ——
const agreements = [
  { agreementId: 'AGR-USER', type: 'userAgreement', typeLabel: '用户协议', version: 'v1.1', effectiveAt: '2025-01-01', active: true, clauses: [
    '本平台为宽带装维全流程自助服务平台，用户使用前须完成实名认证。',
    '套餐资费以办理时页面展示为准，合约期内退订须按合约约定承担相应责任。',
    '用户应保证所提交地址、联系方式真实有效，用于资源核查与装维上门。',
    '缴费、充值、报障等操作记录与核心系统一致，用户可随时查询。',
    '平台对关键业务（余额/到期/停机/复机/账单/故障）按通知订阅设置推送消息。'] },
  { agreementId: 'AGR-PRIVACY', type: 'privacyPolicy', typeLabel: '隐私政策', version: 'v1.0', effectiveAt: '2025-01-01', active: true, clauses: [
    '收集范围：手机号、实名信息、家庭地址、联系方式，仅用于业务办理与装维服务。',
    '不向第三方出售或泄露用户个人信息，法律法规另有规定除外。',
    '用户可随时在「账号安全」中查看与维护实名信息。',
    '审计日志保留 36 个月，用户关键操作留痕可追溯。'] },
];

// —— 余额(balances)与充值面额(topupDenominations) ——
const balances = [
  { customerId: 1, balance: 42, lowThreshold: 50, updatedAt: '2025-08-17 07:30' },
];
const topupDenominations = [
  { denomId: 'D-50', amount: 50 }, { denomId: 'D-100', amount: 100 }, { denomId: 'D-200', amount: 200 },
];

// —— 电子发票(userInvoices): 仅已缴账期可开(口径见 DATA-ALIGNMENT §2) ——
const userInvoices = [
  { invoiceId: 'INV-202507-001', customerId: 1, period: '2025-07', billNo: 'BILL-202507', amount: 158, titleType: '个人', title: '王先生', taxNo: null, issuedAt: '2025-07-26', pdfUrl: '#' },
];

// —— 投诉(userComplaints): 计费争议等非报障类,bizType 区分 ——
const userComplaints = [
  { complaintId: 'CP-001', customerId: 1, type: 'billing', typeLabel: '计费争议 · 2025-07 账期', relOrderNo: '', description: '', status: 'RESOLVED', statusLabel: '已解决', createdAt: '2025-07-28', handler: '客服·陈客服' },
];

// —— 实名核验记录(userVerifyRecords): 客户实名档案的过程留痕 ——
const userVerifyRecords = [
  { customerId: 1, method: '证件 + 人像比对', time: '2023-05-11 10:12', result: 'PASS' },
  { customerId: 1, method: '证件 OCR', time: '2023-05-11 10:08', result: 'PASS' },
];

// —— 产品卖点配置(productSpecs): 商品详情页规格项,admin 可配 ——
const productSpecs = [
  { productId: 'P-1000', specs: [
    { label: '安装费', value: '首装 ¥0' },
    { label: '光猫 / 路由器', value: '含设备 · 押金 ¥0' },
    { label: 'IPTV', value: '200+ 频道' },
    { label: '合约期', value: '24 个月 · 到期自动续约' }] },
];

// —— 账单明细项(userBillItems): billNo 外键 db.bills ——
const userBillItems = [
  { billNo: 'BILL-202508', name: '1000M 极速宽带套餐费', range: '08-01 ~ 08-31', amount: 199 },
  { billNo: 'BILL-202508', name: 'IPTV 高清电视', range: '08-01 ~ 08-31', amount: 10 },
  { billNo: 'BILL-202508', name: '全屋 WiFi', range: '08-01 ~ 08-31', amount: 15 },
  { billNo: 'BILL-202508', name: '合约减免', range: '首年立减', amount: -66 },
];

module.exports = {
  userAccounts, userAddresses, userPlans, addonCatalog, addonSubscriptions, notifyPrefs,
  userFaqs, userMessages, coupons, inviteConfig, usageRecords, diyGuides, agreements,
  balances, topupDenominations, userInvoices, userComplaints, userVerifyRecords,
  productSpecs, userBillItems,
};
