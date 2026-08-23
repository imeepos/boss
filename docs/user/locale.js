// 用户端 · 共享多语言资源（zh-CN/en-US/ms-MY）。
// 用法：在页面引入 locale.js 后，调用 window.L.t('key') 获取当前语言翻译。
// 语言通过 localStorage 'boss_user_locale' 切换。
// 所有页面的 data-i18n 属性会自动翻译。
(function (global) {
  'use strict';

  var LOCALES = {
    'zh-CN': {
      // 通用
      'nav.back': '返回',
      'nav.profile': '我的',
      'nav.home': '首页',
      'nav.products': '产品',
      'nav.orders': '订单',
      'nav.points': '积分',
      'lang.name': '中文',
      // 错误
      'errors.load': '暂时无法加载，请稍后重试。',
      'errors.network': '网络错误，请检查网络连接。',
      'errors.auth': '请先登录。',
      'errors.empty': '暂无记录。',
      // 状态
      'status.active': '生效中',
      'status.expired': '已过期',
      'status.pending': '处理中',
      'status.done': '已完成',
      'status.cancelled': '已取消',
      'status.paid': '已支付',
      'status.unpaid': '未支付',
      // 订单
      'order.title': '订单详情',
      'order.list': '我的订单',
      'order.no': '订单号',
      'order.status': '状态',
      'order.product': '产品',
      'order.address': '地址',
      'order.stage': '当前环节',
      'order.submit': '提交订单',
      'order.cancel': '取消订单',
      // 产品
      'product.title': '产品套餐',
      'product.detail': '套餐详情',
      'product.monthly': '月费',
      'product.buy': '立即办理',
      // 账单
      'bill.title': '我的账单',
      'bill.detail': '账单明细',
      'bill.pay': '在线缴费',
      'bill.amount': '金额',
      'bill.due': '到期日',
      'bill.paid': '已缴',
      'bill.unpaid': '待缴',
      // 支付
      'pay.title': '在线缴费',
      'pay.success': '支付成功',
      'pay.fail': '支付失败',
      'pay.result': '支付结果',
      // 工单
      'fault.title': '故障报修',
      'fault.detail': '报修详情',
      'fault.submit': '提交报修',
      'fault.desc': '故障描述',
      // 积分
      'points.title': '积分中心',
      'points.balance': '当前积分',
      'points.tasks': '积分任务',
      'points.entries': '积分流水',
      'points.done': '已完成',
      'points.complete': '完成任务',
      // 优惠券
      'coupon.title': '优惠券与活动',
      'coupon.available': '可用',
      'coupon.used': '已使用',
      'coupon.expired': '已过期',
      // 消息
      'messages.title': '消息中心',
      'messages.empty': '暂无消息',
      // 地址
      'address.title': '家庭地址管理',
      'address.add': '添加地址',
      'address.edit': '编辑地址',
      // 套餐
      'myplan.title': '我的套餐',
      'myplan.change': '改套餐',
      // 安全
      'security.title': '账号安全',
      'security.password': '修改密码',
      // 评价
      'rate.title': '服务评价',
      'rate.submit': '提交评价',
      // 充值
      'topup.title': '余额充值',
      // 用量
      'usage.title': '网络用量',
      // 通知
      'notify.title': '通知订阅设置',
      // 迁址
      'move.title': '迁址移机',
      // 退订
      'cancel.title': '退订拆机',
      // 帮助
      'help.title': '帮助中心',
      // 客服
      'service.title': '在线客服',
      // 协议
      'agreement.title': '用户协议与隐私政策',
      // 发票
      'invoice.title': '电子发票',
      // 凭证
      'receipt.title': '缴费凭证',
      // 实名
      'verify.title': '实名认证',
    },
    'en-US': {
      'nav.back': 'Back',
      'nav.profile': 'Profile',
      'nav.home': 'Home',
      'nav.products': 'Products',
      'nav.orders': 'Orders',
      'nav.points': 'Points',
      'lang.name': 'English',
      'errors.load': 'Unable to load. Try again later.',
      'errors.network': 'Network error. Check your connection.',
      'errors.auth': 'Please sign in first.',
      'errors.empty': 'No records.',
      'status.active': 'Active',
      'status.expired': 'Expired',
      'status.pending': 'Processing',
      'status.done': 'Completed',
      'status.cancelled': 'Cancelled',
      'status.paid': 'Paid',
      'status.unpaid': 'Unpaid',
      'order.title': 'Order Details',
      'order.list': 'My Orders',
      'order.no': 'Order No.',
      'order.status': 'Status',
      'order.product': 'Product',
      'order.address': 'Address',
      'order.stage': 'Current Stage',
      'order.submit': 'Submit Order',
      'order.cancel': 'Cancel Order',
      'product.title': 'Products',
      'product.detail': 'Plan Details',
      'product.monthly': 'Monthly',
      'product.buy': 'Subscribe Now',
      'bill.title': 'My Bills',
      'bill.detail': 'Bill Details',
      'bill.pay': 'Pay Online',
      'bill.amount': 'Amount',
      'bill.due': 'Due Date',
      'bill.paid': 'Paid',
      'bill.unpaid': 'Unpaid',
      'pay.title': 'Pay Online',
      'pay.success': 'Payment Successful',
      'pay.fail': 'Payment Failed',
      'pay.result': 'Payment Result',
      'fault.title': 'Report Fault',
      'fault.detail': 'Fault Details',
      'fault.submit': 'Submit Report',
      'fault.desc': 'Fault Description',
      'points.title': 'Points',
      'points.balance': 'Balance',
      'points.tasks': 'Tasks',
      'points.entries': 'History',
      'points.done': 'Completed',
      'points.complete': 'Complete',
      'coupon.title': 'Coupons',
      'coupon.available': 'Available',
      'coupon.used': 'Used',
      'coupon.expired': 'Expired',
      'messages.title': 'Messages',
      'messages.empty': 'No messages',
      'address.title': 'Addresses',
      'address.add': 'Add Address',
      'address.edit': 'Edit Address',
      'myplan.title': 'My Plan',
      'myplan.change': 'Change Plan',
      'security.title': 'Security',
      'security.password': 'Change Password',
      'rate.title': 'Rate Service',
      'rate.submit': 'Submit Rating',
      'topup.title': 'Top Up',
      'usage.title': 'Usage',
      'notify.title': 'Notifications',
      'move.title': 'Relocate',
      'cancel.title': 'Cancel Service',
      'help.title': 'Help Center',
      'service.title': 'Customer Service',
      'agreement.title': 'Terms & Privacy',
      'invoice.title': 'Invoice',
      'receipt.title': 'Receipt',
      'verify.title': 'Identity Verification',
    },
    'ms-MY': {
      'nav.back': 'Kembali',
      'nav.profile': 'Profil',
      'nav.home': 'Utama',
      'nav.products': 'Produk',
      'nav.orders': 'Pesanan',
      'nav.points': 'Mata ganjaran',
      'lang.name': 'Melayu',
      'errors.load': 'Tidak dapat dimuatkan. Cuba lagi.',
      'errors.network': 'Ralat rangkaian. Semak sambungan anda.',
      'errors.auth': 'Sila log masuk dahulu.',
      'errors.empty': 'Tiada rekod.',
      'status.active': 'Aktif',
      'status.expired': 'Tamat tempoh',
      'status.pending': 'Dalam proses',
      'status.done': 'Selesai',
      'status.cancelled': 'Dibatalkan',
      'status.paid': 'Dibayar',
      'status.unpaid': 'Belum dibayar',
      'order.title': 'Butiran Pesanan',
      'order.list': 'Pesanan Saya',
      'order.no': 'No. Pesanan',
      'order.status': 'Status',
      'order.product': 'Produk',
      'order.address': 'Alamat',
      'order.stage': 'Peringkat Semasa',
      'order.submit': 'Hantar Pesanan',
      'order.cancel': 'Batalkan Pesanan',
      'product.title': 'Produk',
      'product.detail': 'Butiran Pelan',
      'product.monthly': 'Bulanan',
      'product.buy': 'Langgan Sekarang',
      'bill.title': 'Bil Saya',
      'bill.detail': 'Butiran Bil',
      'bill.pay': 'Bayar Dalam Talian',
      'bill.amount': 'Jumlah',
      'bill.due': 'Tarikh Akhir',
      'bill.paid': 'Dibayar',
      'bill.unpaid': 'Belum dibayar',
      'pay.title': 'Bayar Dalam Talian',
      'pay.success': 'Pembayaran Berjaya',
      'pay.fail': 'Pembayaran Gagal',
      'pay.result': 'Keputusan Pembayaran',
      'fault.title': 'Laporkan Kerosakan',
      'fault.detail': 'Butiran Kerosakan',
      'fault.submit': 'Hantar Laporan',
      'fault.desc': 'Penerangan Kerosakan',
      'points.title': 'Mata ganjaran',
      'points.balance': 'Baki mata',
      'points.tasks': 'Tugasan',
      'points.entries': 'Sejarah',
      'points.done': 'Selesai',
      'points.complete': 'Lengkapkan',
      'coupon.title': 'Kupon',
      'coupon.available': 'Tersedia',
      'coupon.used': 'Digunakan',
      'coupon.expired': 'Tamat tempoh',
      'messages.title': 'Mesej',
      'messages.empty': 'Tiada mesej',
      'address.title': 'Alamat',
      'address.add': 'Tambah Alamat',
      'address.edit': 'Sunting Alamat',
      'myplan.title': 'Pelan Saya',
      'myplan.change': 'Tukar Pelan',
      'security.title': 'Keselamatan',
      'security.password': 'Tukar Kata Laluan',
      'rate.title': 'Nilai Perkhidmatan',
      'rate.submit': 'Hantar Penilaian',
      'topup.title': 'Tambah Nilai',
      'usage.title': 'Penggunaan',
      'notify.title': 'Pemberitahuan',
      'move.title': 'Pindah Lokasi',
      'cancel.title': 'Batalkan Perkhidmatan',
      'help.title': 'Pusat Bantuan',
      'service.title': 'Perkhidmatan Pelanggan',
      'agreement.title': 'Terma & Privasi',
      'invoice.title': 'Invois',
      'receipt.title': 'Resit',
      'verify.title': 'Pengesahan Identiti',
    }
  };

  function getLocale() {
    return localStorage.getItem('boss_user_locale') || 'zh-CN';
  }

  function setLocale(lang) {
    localStorage.setItem('boss_user_locale', lang);
    location.reload();
  }

  function t(key) {
    var lang = getLocale();
    var dict = LOCALES[lang] || LOCALES['zh-CN'];
    return dict[key] || key;
  }

  function translatePage() {
    document.querySelectorAll('[data-i18n]').forEach(function (el) {
      var key = el.getAttribute('data-i18n');
      var val = t(key);
      if (val !== key) {
        if (el.tagName === 'INPUT' && el.placeholder) {
          el.placeholder = val;
        } else {
          el.textContent = val;
        }
      }
    });
  }

  // 自动翻译页面
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', translatePage);
  } else {
    translatePage();
  }

  // 暴露 API
  global.L = {
    t: t,
    locale: getLocale,
    setLocale: setLocale,
    translatePage: translatePage,
    LOCALES: LOCALES
  };
})(window);
