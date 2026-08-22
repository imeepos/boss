# 0002-2026-08-22-用户端配置-ID-undefined

## 现象

后台管理 → 客户与资费 → 用户端配置（`/bss/userdata`）7 个 Tab 的 ID 列全部显示 `undefined`，
操作列点击 toggle/disable 实际请求 `PUT /addons/undefined/toggle` 之类死链（404），
通知设置 Tab 的名称列显示 `—`、状态列错为 `停用`（语义本应是偏好开关，不是状态枚举）。

## 根因

`web/admin/src/pages/bss/userdata/tabs.ts` 给所有行强认 `Row.id` 字段，但后端 SQL 实际别名因表而异：

| Tab | 后端 SQL 别名（pg_lists.go） | 前端假设 | 实际 |
|---|---|---|---|
| notify | `customer_id AS "customerId"` | `id` | `customerId` |
| addons | `addon_id AS "addonId"` | `id` | `addonId` |
| coupons | `coupon_id AS "couponId"` | `id` | `couponId` |
| topup | `denom_id AS "denomId"` | `id` | `denomId` |
| faqs | `faq_id AS "faqId"` | `id` | `faqId` |
| guides | `guide_id AS "guideId"` | `id` | `guideId` |
| invite | `id` | `id` | `id` ✓ |

唯一命中的是 invite（也是它先被加上 TABS，肉眼自测很容易忽略前面 6 个全错）。

## 暴露链

- 前端 `index.tsx`：`String(r.id)` 渲染 ID、`row.id` 拼动作 URL、`String(r.id)` 作 React key，
  三个地方全部 undefined，没有运行期报错（React 默认容忍 undefined key，URL 拼接为 `undefined` 字符串）。
- 名称列 fallback：`r.name ?? r.title ?? r.question ?? r.code ?? r.label`，notify 字段是
  `customerName`，全部不命中 → `—`。
- 状态列 fallback：`r.status ?? (r.enabled ? on : off) ?? '—'`，topup/notify 无 status/enabled，
  → `—` / 错语义。

## 修复（commit cffa226）

- `tabs.ts`：`Row` 改为开放 map；`TabDef` 新增 `idKey: string`，每个 Tab 标注后端实际主键列名。
- `index.tsx`：`String(r[cur.idKey])`、`row[def.idKey]` 拼 URL、key 一并改。
- 名称列 fallback 链追加 `r.customerName`；状态列 fallback 链追加 `(r.active ? on : off)`。
- 新增 `tabs.test.ts`：锁住 7 个 Tab 的 idKey 映射，防止再次漂移。

## 后续未做（本 issue 范围外）

- TopupDenomination 的名称列仍显示 `—`：应展示金额（amount/100 = 元）。
- NotifySetting 的状态列错为 `停用`：语义本就是偏好开关，不是状态，应改两栏（业务/营销）
  或彻底改为偏好表单。当前属于"按用户习惯改为主表+明细抽屉"的体验改造。
- 其他列表型配置页（用户列表、客户档案、产品资费…）可能也有同类漂移——下次同类任务前先
  全量 grep `pg_lists.go` 列名与前端 `r.id`/`row.id` 用法。

## 教训

列表型配置页前端不要给所有 Tab 假设一个统一 `id` 主键列。SQL `AS "..."` 的别名由各表
独立声明，前后端字段名约定必须有显式对账层（test/契约 yaml），不能依赖肉眼自测。