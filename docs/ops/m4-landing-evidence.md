# M4 3.4 官网获客闭环 · 验收证据

> 日期：2026-08-28｜对应 `docs/plan/q4-launch-growth-plan.md` 方向三 3.4
> 环境：102 真实 API + 迁移 000161 已部署，无 mock。

## 1. 注册来源追踪（本批代码，commit 7f483ba8）

- migration **000161**：`customer_registrations.source VARCHAR(32) DEFAULT ''`（已部署实测列存在）。
- 契约：user `POST /customer-registrations` 增可选 `source`（≤32）；admin 审核队列列表自动带出。
- 用途：官网/落地页带 source 提交注册 → 转化可归因（如 `landing-m4`）。

## 2. 真实链路验证（2026-08-28）

1. 带 source 注册：`POST /api/user/v1/customer-registrations`{...,"source":"landing-m4"} →
   code:0，id=13，PENDING（公开端点免登录）。
2. admin 队列：`GET /customer-registrations?status=PENDING` → id=13 source='landing-m4' 可见。
3. CMS 活动文章：`POST /site-posts`（slug=m4-first-order-campaign，PUBLISHED，id=8）→
   公开读 `GET /api/admin/v1/site/posts` 免鉴权返回全文（官网首页消费同一 API）。

## 3. 闭环口径（3.4 达成）

官网(CMS 公开文章,含活动内容) → 客户带 source 注册(落地归因) → admin 审核队列按 source 追踪
→ 审核开户后进入 M4 活动（首单券/积分）核销链路 → coupon/points recon 对账。
获客 → 转化 → 活动消费全链在真实环境可追踪。

未含（如实记录）：官网静态站本身不在本仓（nginx 代理 + CMS API 供数），落地页 HTML 由站点侧
消费公开 API 即可；utm 多级参数 v1 收敛为单 source 字段。

## 4. M4 总体结论

- 3.2 营销活动（双活动 + 发放/兑换/核销/对账全链）✅
- 3.3 首单风控（渠道 + 直营双链路，可配可审计）✅
- 3.4 官网获客闭环（来源追踪 + CMS 活动 + 公开读）✅
- 老带新推荐码：需推荐关系建模，移入 M4 后段/下季评估（见计划 §5 红线：不铺大域）。
- **M4 里程碑达成**（三项验收全部真实环境闭环）。

## 5. 全计划进度

✅ M0 守夜　✅ M1 支付/对账　✅ M2 worker Android 收口（真机走查待设备）　🟡 M3 渠道（A/B/C 待拍板）　✅ **M4 营销+风控+获客**　⬜ M5 放量总验收