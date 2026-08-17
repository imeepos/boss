// 假数据 —— GIS/经营分析/报告(契约: api/openapi/admin/intel.yaml)。
'use strict';

const ok = { code: 0, message: 'success' };

// GIS 同步仅发生在订单第 12 环节(更新GIS)之后,未完成订单一律"待同步"
const addressLinks = [
  { addressCode: 'A-1-101', pathLabel: '望京Y · 1栋 · 101', orderNo: 'ORD-20250816-018', epcCode: 'EPC-0003', syncStatus: '已同步' },
  { addressCode: 'A-3-501', pathLabel: '望京X · 3栋 · 501', orderNo: 'ORD-20250817-001', epcCode: 'EPC-0001', syncStatus: '待同步' },
  { addressCode: 'A-12-906', pathLabel: '望京X · 12栋 · 906', orderNo: 'ORD-20250817-003', epcCode: 'EPC-0012', syncStatus: '待同步' },
];

const kpis = [
  { key: 'portUsage', label: '端口利用率', value: '72.3%' },
  { key: 'installRate', label: '装机转化率', value: '41.8%' },
  { key: 'maintainCost', label: '单用户维护成本', value: '¥6.4/月' },
  { key: 'assetHealth', label: '资产健康度评分', value: '86.2' },
  { key: 'regionRoi', label: '区域投资回报率', value: '23.5%' },
];

const REGIONS = ['吕宋大区', '比萨扬大区', '棉兰老大区'];
const matrixRows = [
  { brand: 'LEG-A 主品牌·企业', values: { '吕宋大区': '¥8.2M', '比萨扬大区': '—', '棉兰老大区': '—' }, total: '¥8.2M' },
  { brand: 'LEG-B 家庭宽带', values: { '吕宋大区': '¥5.4M', '比萨扬大区': '—', '棉兰老大区': '¥3.1M' }, total: '¥8.5M' },
  { brand: 'LEG-C 批发品牌', values: { '吕宋大区': '—', '比萨扬大区': '¥2.7M', '棉兰老大区': '—' }, total: '¥2.7M' },
];
const matrixTotal = { brand: '合计', values: { '吕宋大区': '¥13.6M', '比萨扬大区': '¥2.7M', '棉兰老大区': '¥3.1M' }, total: '¥19.4M' };

const replaceList = [
  { device: 'OLT-15', health: 41, faultCount: 23, years: '6 年', suggestion: '立即替换' },
  { device: '分光器-208', health: 55, faultCount: 12, years: '5 年', suggestion: '优先级高' },
  { device: 'OLT-33', health: 67, faultCount: 8, years: '4 年', suggestion: '观察' },
];

const reports = [
  { reportId: 1, title: '2025-08-17 经营日报', period: '日', generatedAt: '08:00', audience: '经营分析组', status: '已推送' },
  { reportId: 2, title: '2025-08 第3周 周报', period: '周', generatedAt: '周一 09:00', audience: '经营分析组/管理层', status: '已推送' },
  { reportId: 3, title: '2025-08 月度报告', period: '月', generatedAt: '—', audience: '管理层', status: '生成中' },
];

// 大区筛选按前缀匹配(吕宋/吕宋大区均可)
function hitRegion(name, region) {
  return !region || name.indexOf(region.replace(/大区$/, '')) === 0;
}

module.exports = {
  'GET /gis/address-links': { items: addressLinks },

  'GET /analytics/overview': ({ query }) => {
    const brand = query.brand || '';
    const region = query.region || '';
    const rows = matrixRows.filter((r) => !brand || r.brand === brand);
    const regions = REGIONS.filter((r) => hitRegion(r, region));
    return {
      kpis: kpis,
      brandRegionMatrix: { regions: regions, rows: rows.concat([matrixTotal]) },
      replaceList: replaceList,
    };
  },

  'GET /reports': ({ query }) => {
    const kw = query.keyword || '';
    return { items: reports.filter((r) => r.title.indexOf(kw) >= 0) };
  },
  'POST /reports': ok,
  'POST /reports/{reportId}/send': ({ params }) =>
    Object.assign({}, ok, { reportId: Number(params.reportId) }),
};
