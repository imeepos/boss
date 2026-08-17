// mock data —— 用户端假数据。字段对齐 api/openapi/user.yaml 组件 schema。
// JSON 字段统一 lowerCamelCase(见 docs/contract/fields.md 第 0 节)。
module.exports = {
  profile: {
    customerId: 1,
    name: '王先生',
    phoneMasked: '138****1234',
    realName: {
      nameMasked: '王**',
      idType: '身份证',
      idNoMasked: '110***********1234',
      status: 'VERIFIED',
    },
    addresses: [
      {
        addressId: 'ADDR-001',
        label: '望京X · 3栋 · 501',
        isDefault: true,
        contact: '王先生',
        phoneMasked: '138****1234',
        community: '望京X',
        building: '3栋',
        door: '501',
      },
      {
        addressId: 'ADDR-002',
        label: '望京Y · 1栋 · 101',
        isDefault: false,
        contact: '王先生',
        phoneMasked: '138****1234',
        community: '望京Y',
        building: '1栋',
        door: '101',
      },
    ],
    plan: {
      planId: 'PLAN-1000',
      name: '1000M 极速宽带',
      monthlyFee: 199,
      contractEnd: '2026-08',
      status: 'ACTIVE',
      installAddress: '望京X · 3栋 · 501',
    },
  },

  security: {
    realNameStatus: 'VERIFIED',
    nameMasked: '王**',
    idNoMasked: '110***********1234',
    verifyAt: '2023-05-11',
    passwordUpdatedAt: '2025-01-03',
    phoneMasked: '138****1234',
  },

  verify: {
    status: 'VERIFIED',
    nameMasked: '王**',
    idNoMasked: '110***********1234',
    verifyAt: '2023-05-11',
    records: [
      { method: '证件 + 人像比对', time: '2023-05-11 10:12', result: 'PASS' },
      { method: '证件 OCR', time: '2023-05-11 10:08', result: 'PASS' },
    ],
  },

  notifySettings: {
    business: {
      bill: true,
      suspendResume: true,
      faultNotice: true,
      installProgress: true,
    },
    marketing: {
      promo: true,
      planRecommend: false,
    },
    channels: {
      inApp: true,
      sms: true,
      push: true,
    },
  },

  products: {
    items: [
      {
        productId: 'P-300',
        category: 'broadband',
        name: '300M 畅享宽带',
        bandwidth: '300M',
        monthlyFee: 99,
        description: '下行 300M · 含光猫 · 合约 12 个月',
        contractMonths: 12,
        featured: false,
      },
      {
        productId: 'P-500',
        category: 'broadband',
        name: '500M 畅享宽带',
        bandwidth: '500M',
        monthlyFee: 129,
        description: '下行 500M · 含光猫 + 路由器 · 合约 12 个月',
        contractMonths: 12,
        featured: false,
      },
      {
        productId: 'P-1000',
        category: 'broadband',
        name: '1000M 极速宽带',
        bandwidth: '1000M',
        monthlyFee: 199,
        description: '下行 1000M · 含光猫 + 路由器 + IPTV · 合约 24 个月',
        contractMonths: 24,
        featured: true,
      },
    ],
    addons: [
      { addonId: 'A-IPTV', name: 'IPTV 高清电视', monthlyFee: 10, description: '¥10/月 · 200+ 频道', subscribed: false },
      { addonId: 'A-WIFI', name: '全屋 WiFi', monthlyFee: 15, description: '¥15/月 · 信号覆盖', subscribed: false },
    ],
  },

  productDetail: {
    product: {
      productId: 'P-1000',
      category: 'broadband',
      name: '1000M 极速宽带',
      bandwidth: '1000M',
      monthlyFee: 199,
      description: '下行 1000M / 上行 50M · 含光猫 + 路由器 + IPTV · 合约 24 个月',
      contractMonths: 24,
      featured: true,
    },
    specs: [
      { label: '安装费', value: '首装 ¥0' },
      { label: '光猫 / 路由器', value: '含设备 · 押金 ¥0' },
      { label: 'IPTV', value: '200+ 频道' },
      { label: '合约期', value: '24 个月 · 到期自动续约' },
    ],
    compare: [
      { productId: 'P-300', category: 'broadband', name: '300M 畅享宽带', bandwidth: '300M', monthlyFee: 99, description: '¥99/月 · 含光猫', contractMonths: 12, featured: false },
      { productId: 'P-500', category: 'broadband', name: '500M 畅享宽带', bandwidth: '500M', monthlyFee: 129, description: '¥129/月 · 光猫 + 路由器', contractMonths: 12, featured: false },
    ],
  },

  addons: {
    available: [
      { addonId: 'A-IPTV', name: 'IPTV 高清电视', monthlyFee: 10, description: '¥10/月 · 200+ 频道', subscribed: false },
      { addonId: 'A-WIFI', name: '全屋 WiFi', monthlyFee: 15, description: '¥15/月 · 信号覆盖', subscribed: false },
      { addonId: 'A-CLOUD', name: '云盘存储', monthlyFee: 8, description: '¥8/月 · 500GB', subscribed: false },
      { addonId: 'A-CAM', name: '家庭安全看护', monthlyFee: 20, description: '¥20/月 · 摄像头云存储', subscribed: false },
    ],
    subscribed: [
      { addonId: 'A-IPTV-OLD', name: 'IPTV 高清电视', monthlyFee: 10, description: '2023-05-11 订购 · ¥10/月', subscribed: true },
      { addonId: 'A-WIFI-OLD', name: '全屋 WiFi', monthlyFee: 15, description: '2025-02-01 订购 · ¥15/月', subscribed: true },
    ],
  },

  orders: {
    items: [
      {
        orderNo: 'ORD-20250817-001',
        status: 'INSTALLING',
        statusLabel: '装维中',
        productName: '1000M 极速宽带',
        address: '望京X · 3栋501',
        stage: 9,
        stageLabel: '扫码绑定',
        estimateFinish: '08-17 完成',
        canRate: false,
      },
      {
        orderNo: 'ORD-20250817-002',
        status: 'RESERVED',
        statusLabel: '已预占',
        productName: '500M 畅享宽带',
        address: '望京X · 5栋302',
        stage: 3,
        stageLabel: '端口预占',
        estimateFinish: null,
        canRate: false,
      },
      {
        orderNo: 'ORD-20250816-018',
        status: 'DONE',
        statusLabel: '已完成',
        productName: '300M 畅享宽带',
        address: '望京Y · 1栋101',
        stage: 12,
        stageLabel: '更新 GIS 地图',
        estimateFinish: null,
        canRate: true,
      },
    ],
  },

  orderDetail: {
    order: {
      orderNo: 'ORD-20250817-001',
      status: 'INSTALLING',
      statusLabel: '装维中',
      productName: '1000M 极速宽带 · ¥199/月',
      address: '望京X · 3栋 · 501',
      stage: 9,
      stageLabel: '扫码绑定',
      canRate: false,
    },
    submitedAt: '2025-08-17 09:02',
    technicianName: '张师傅',
    technicianPhoneMasked: '138****8899',
    completedStage: 8,
    timeline: [
      { stage: 1, title: '用户下单', result: 'DONE', meta: '08-17 09:02 · 完成' },
      { stage: 2, title: '资源核查', result: 'DONE', meta: '08-17 09:05 · 有空闲端口 · 3m' },
      { stage: 3, title: '端口预占', result: 'DONE', meta: '08-17 09:09 · 4m' },
      { stage: 4, title: '合同收费', result: 'DONE', meta: '08-17 09:11 · 已收款 ¥199 · 2m' },
      { stage: 5, title: '标签预绑定', result: 'DONE', meta: '08-17 09:12 · 标签 EPC-0001 · 3m' },
      { stage: 6, title: '创建认证账号', result: 'DONE', meta: '08-17 09:15 · 3m' },
      { stage: 7, title: '预下发配置', result: 'DONE', meta: '08-17 09:18 · 3m' },
      { stage: 8, title: '派单', result: 'DONE', meta: '08-17 09:20 · 张师傅已接单 · 2m' },
      { stage: 9, title: '扫码绑定', result: 'DOING', meta: '进行中 · 师傅现场操作' },
      { stage: 10, title: '激活用户', result: 'PENDING', meta: '待处理' },
      { stage: 11, title: '激活回调', result: 'PENDING', meta: '待处理' },
      { stage: 12, title: '更新 GIS 地图', result: 'PENDING', meta: '待处理' },
    ],
  },

  bills: {
    currentDue: 158,
    currentPeriod: '2025-08',
    items: [
      { billNo: 'BILL-202508', period: '2025-08', productName: '1000M 极速宽带', periodRange: '08-01 ~ 08-31', amount: 158, status: 'UNPAID', statusLabel: '未缴' },
      { billNo: 'BILL-202507', period: '2025-07', productName: '1000M 极速宽带', periodRange: '07-01 ~ 07-31', amount: 158, status: 'PAID', statusLabel: '已缴' },
      { billNo: 'BILL-202506', period: '2025-06', productName: '500M 畅享宽带', periodRange: '06-01 ~ 06-30', amount: 129, status: 'PAID', statusLabel: '已缴' },
    ],
  },

  billDetail: {
    bill: { billNo: 'BILL-202508', period: '2025-08', productName: '1000M 极速宽带', periodRange: '08-01 ~ 08-31', amount: 158, status: 'UNPAID', statusLabel: '未缴' },
    items: [
      { name: '1000M 极速宽带套餐费', range: '08-01 ~ 08-31', amount: 199 },
      { name: 'IPTV 高清电视', range: '08-01 ~ 08-31', amount: 10 },
      { name: '全屋 WiFi', range: '08-01 ~ 08-31', amount: 15 },
      { name: '合约减免', range: '首年立减', amount: -66 },
    ],
    totalDue: 158,
    autoPayEnabled: false,
  },

  payments: {
    items: [
      { payNo: 'PAY20250725001', amount: 158, period: '2025-07', payMethod: '微信支付', paidAt: '2025-07-25 10:12' },
      { payNo: 'PAY20250624001', amount: 129, period: '2025-06', payMethod: '支付宝', paidAt: '2025-06-24 09:40' },
    ],
  },

  receipts: {
    'PAY20250817001': {
      receiptNo: 'OR-20250817-0001',
      customerName: '王先生',
      phoneMasked: '138****1234',
      amount: 158,
      period: '2025-08',
      payMethod: '微信支付',
      paidAt: '2025-08-17 10:12',
      payNo: 'PAY20250817001',
    },
    'PAY20250725001': {
      receiptNo: 'OR-20250725-0001',
      customerName: '王先生',
      phoneMasked: '138****1234',
      amount: 158,
      period: '2025-07',
      payMethod: '微信支付',
      paidAt: '2025-07-25 10:12',
      payNo: 'PAY20250725001',
    },
    'PAY20250624001': {
      receiptNo: 'OR-20250624-0001',
      customerName: '王先生',
      phoneMasked: '138****1234',
      amount: 129,
      period: '2025-06',
      payMethod: '支付宝',
      paidAt: '2025-06-24 09:40',
      payNo: 'PAY20250624001',
    },
  },

  balance: {
    balance: 42,
    denominations: [50, 100, 200],
  },

  invoice: {
    titleType: '个人',
    title: '王先生',
    taxNo: null,
    availablePeriods: [
      { billNo: 'BILL-202508', period: '2025-08', productName: '1000M 极速宽带', periodRange: '08-01 ~ 08-31', amount: 158, status: 'PAID', statusLabel: '已缴' },
      { billNo: 'BILL-202507', period: '2025-07', productName: '1000M 极速宽带', periodRange: '07-01 ~ 07-31', amount: 158, status: 'PAID', statusLabel: '已缴' },
    ],
    records: [
      { period: '2025-07', amount: 158, issuedAt: '2025-07-26', pdfUrl: '#' },
    ],
  },

  faults: {
    items: [
      { ticketNo: 'TKT-20250817-012', faultType: 'no_internet', faultTypeLabel: '单户断网（紧急 SLA）', address: '望京X · 10栋 · 1801', createdAt: '2025-08-17 09:40', status: 'PROCESSING', statusLabel: '处理中' },
      { ticketNo: 'TKT-20250730-005', faultType: 'slow', faultTypeLabel: '网速慢', address: '望京X · 3栋 · 501', createdAt: '2025-07-30 15:02', status: 'RESOLVED', statusLabel: '已解决' },
    ],
  },

  faultDetail: {
    fault: { ticketNo: 'TKT-20250817-012', faultType: 'no_internet', faultTypeLabel: '单户断网（紧急 SLA≤4h）', address: '望京X · 10栋 · 1801', createdAt: '2025-08-17 09:40', status: 'PROCESSING', statusLabel: '处理中' },
    technicianName: '张师傅',
    technicianPhoneMasked: '138****7788',
    sla: '≤4h',
    timeline: [
      { step: 1, title: '报障', result: 'DONE', meta: '09:40 · 已自动关联资产/端口' },
      { step: 2, title: '诊断', result: 'DONE', meta: '09:42 · 远程诊断入工单' },
      { step: 3, title: '派单', result: 'DONE', meta: '09:45 · 张师傅已接单 · SLA 计时' },
      { step: 4, title: '修复', result: 'DOING', meta: '进行中 · 到场处理并上报结果' },
      { step: 5, title: '复核', result: 'PENDING', meta: '系统自动验证网络恢复' },
      { step: 6, title: '回访', result: 'PENDING', meta: '满意度调查+回填' },
    ],
  },

  complaints: {
    items: [
      { complaintId: 'CP-001', type: 'billing', typeLabel: '计费争议 · 2025-07 账期', relOrderNo: '', description: '', status: 'RESOLVED', statusLabel: '已解决', createdAt: '2025-07-28' },
    ],
  },

  faq: {
    items: [
      { id: 'faq-1', question: '如何修改套餐？', answer: '进入「我的套餐」→「改套餐」选择新套餐并提交。' },
      { id: 'faq-2', question: '如何开具电子发票？', answer: '进入「电子发票」选择可开票账期申请开票。' },
      { id: 'faq-3', question: '账单怎么看？', answer: '进入「我的账单」查看账单明细与费用构成。' },
      { id: 'faq-4', question: '如何迁址移机？', answer: '进入「我的套餐」→「迁址」，填写新地址提交。' },
      { id: 'faq-5', question: '上网故障如何自检？', answer: '进入「故障报修」→「自助排障」按引导排查。' },
    ],
  },

  home: {
    customerName: '王先生',
    phoneMasked: '138****1234',
    onlineStatus: '服务在线 · 网络正常',
    hasUnread: true,
    plan: { planId: 'PLAN-1000', name: '1000M 家庭宽带', monthlyFee: 199, contractEnd: '2026-08', status: 'ACTIVE', installAddress: '望京X · 3栋 · 501' },
    currentBill: 158,
    balance: 42,
    contractEnd: '2026-08',
    ongoingOrders: [
      { orderNo: 'ORD-20250817-001', status: 'INSTALLING', statusLabel: '装维中', productName: '1000M 安装', address: '望京X · 3栋501', stage: 9, stageLabel: '扫码绑定', canRate: false },
    ],
    services: [
      { name: '宽带上网', desc: 'LOID 认证 · 带宽 1000M', status: 'NORMAL', statusLabel: '正常' },
      { name: 'IPTV 电视', desc: '增值服务 · 高清频道', status: 'NORMAL', statusLabel: '正常' },
    ],
  },

  messages: {
    items: [
      { messageId: 'M-001', category: 'balance', title: '余额预警', content: '账户余额 ¥5.00 低于 ¥50，请及时充值避免停机', tag: '预警', tagLevel: 'balance', createdAt: '2025-08-17', read: false },
      { messageId: 'M-002', category: 'balance', title: '到期停机提醒', content: '账户余额不足，将于明日到期停机，充值后自动复机', tag: '停机', tagLevel: 'balance', createdAt: '2025-08-17', read: false },
      { messageId: 'M-003', category: 'billing', title: '8 月账单已出账', content: '应缴 ¥158.00 · 账期 08-01 ~ 08-31', tag: '未缴', tagLevel: 'bill', createdAt: '2025-08-17', read: false },
      { messageId: 'M-004', category: 'billing', title: '装维进度更新', content: '订单 ORD-20250817-001 已进入「扫码绑定」环节', tag: '新', tagLevel: 'info', createdAt: '2025-08-17', read: false },
      { messageId: 'M-005', category: 'fault', title: '故障公告', content: '望京X 片区 8-17 02:00~04:00 计划割接，可能短暂中断', tag: '公告', tagLevel: 'fault', createdAt: '2025-08-17', read: true },
      { messageId: 'M-006', category: 'promo', title: '夏日宽带优惠', content: '1000M 套餐首月立减 ¥40，点击领取', tag: '活动', tagLevel: 'promo', createdAt: '2025-08-17', read: true },
    ],
  },

  coupons: {
    items: [
      { couponId: 'CPN-001', amount: 40, threshold: 100, title: '1000M 套餐首月立减', expireAt: '2025-08-31', status: 'available' },
      { couponId: 'CPN-002', amount: 20, threshold: 200, title: '余额充值满 200 减 20', expireAt: '2025-09-30', status: 'available' },
    ],
    inviteLink: 'https://boss.example.com/invite/8f3a',
  },

  usage: {
    periodLabel: '本月已用流量（08-01 ~ 08-31）',
    used: 286.4,
    unit: 'GB',
    quota: 1000,
    percent: 29,
    dailyAvg: 9.2,
    forecastRemain: 698,
    detail: { down: 248.1, up: 38.3, iptvNote: '不计入流量' },
    online: { duration: '3 天 14 小时', lastOnlineAt: '2025-08-17 07:41' },
  },

  diySteps: {
    items: [
      {
        id: 's1',
        title: '无法上网',
        desc: '光猫灯异常 / 完全断网',
        steps: [
          { title: '1 检查光猫指示灯', desc: 'LOS 红灯说明光纤未接通，需报修' },
          { title: '2 重启光猫与路由器', desc: '断电 30 秒后重新上电' },
          { title: '3 检查认证状态', desc: '确认账号未欠费停机' },
        ],
      },
      {
        id: 's2',
        title: '网速慢',
        desc: '视频卡顿 / 下载缓慢',
        steps: [
          { title: '1 检查 WiFi 信号', desc: '靠近路由器重测' },
          { title: '2 减少并发大流量', desc: '暂停下载/大流量应用' },
        ],
      },
      {
        id: 's3',
        title: 'IPTV 无信号',
        desc: '电视黑屏 / 无频道',
        steps: [
          { title: '1 检查机顶盒连接', desc: '确认网线与 HDMI 连接' },
          { title: '2 重启机顶盒', desc: '断电重启' },
        ],
      },
      {
        id: 's4',
        title: '光猫告警',
        desc: '红灯 / 闪烁异常',
        steps: [
          { title: '1 检查光纤连接', desc: '确认光纤接头插紧' },
          { title: '2 联系报修', desc: 'LOS 红灯持续需报修' },
        ],
      },
    ],
  },

  agreement: {
    userAgreement: [
      '本平台为宽带装维全流程自助服务平台，用户使用前须完成实名认证。',
      '套餐资费以办理时页面展示为准，合约期内退订须按合约约定承担相应责任。',
      '用户应保证所提交地址、联系方式真实有效，用于资源核查与装维上门。',
      '缴费、充值、报障等操作记录与核心系统一致，用户可随时查询。',
      '平台对关键业务（余额/到期/停机/复机/账单/故障）按通知订阅设置推送消息。',
    ],
    privacyPolicy: [
      '收集范围：手机号、实名信息、家庭地址、联系方式，仅用于业务办理与装维服务。',
      '不向第三方出售或泄露用户个人信息，法律法规另有规定除外。',
      '用户可随时在「账号安全」中查看与维护实名信息。',
      '审计日志保留 36 个月，用户关键操作留痕可追溯。',
    ],
  },
};
