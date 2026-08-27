# 侧边栏菜单按实际内容重划 16 组 + 移除顶栏分组导航（2026-08-27）

## 背景

web/admin 侧边栏 14 组系照抄原型 `docs/admin/menu.js`（13 组）+ partner 组，随页面增长已与实际内容脱节：

- `base 基础配置` 挤了 17 项：系统授权/参数/服务端、六项渠道配置、审计/崩溃日志、导入/备份、账号角色、实名审核混装一组的"回收站"。
- `boss 订单与工单` 挤了 15 项：订单履约（order/dispatch/dismantle/callback）、客服（complaint/feedback/service-metrics）、装维（worker 三兄弟）、内容发布（site/knowledge/release）、消息中心全在一组。
- `alarm 告警中心` 仅 1 项、`aaa 认证计费` 仅 2 项，单薄成组。
- 顶栏 `TopNav` 与侧栏展示同一份分组，双入口冗余；14+ 分组后顶栏横向溢出严重。

## 决策

1. **按页面实际语义重组为 16 组**，排序遵循"总览 → 获客 → 收款 → 履约 → 支撑资源 → 监控 → 触达 → 经营 → 管理底座 → 企业工作台"：
   `overview / bss(收编 realname-review) / billing / boss(改称订单与履约,纯订单+客服) / worker(新组:师傅三兄弟) / ams / oss / provision / quad / aaa(吸并 alarm,改称认证与告警) / cms(新组:官网/知识库/版本发布/消息中心) / intel / org(收编 account、入驻审核) / channel(新组:六项渠道配置+apikey/openplat) / system(新组:授权/参数/服务端/主数据/日志/导入备份) / partner`。
2. **页面 key 与 path 一律不变**：权限码 `menu:<key>`、App.tsx 路由生成、check-contract-sync E 项、fields.md 均不受影响；改的只是分组归属与分组 id。
3. **顶栏分组主导航整体移除**：分组导航职责收归侧栏（可折叠/拖宽/图标），分组上下文由面包屑兜底（Breadcrumb 读 `PAGE_BY_KEY→groupId`）。
4. 角色可见组随拆合同步：ops 增 worker/cms（原 boss 覆盖面拆出），resource_admin 的 alarm → aaa。

## 放弃了什么

- **保持 domain-map.md D 列不动**：D 列"admin 分组"描述的是原型 menu.js 的旧分组，本决策不回填该表；分组语义以 A 列能力域为准，新分组归属见 `web/admin/src/router/menu.def.ts` 头注释。
- **合并 oss+provision、oss+aaa 成"网络运维"大组**：三域（OSS/PROV/AAA+MON）在 domain-map 是独立能力域，运营/资源管理员角色可见性也不同，强合会重蹈 base 大杂烩覆辙。
- **product 挪去 billing**：domain-map 将"产品与资费 PROD"与 CRM 同域（bss），保持不动。
- **顶栏保留精简版分组下拉**：无导航价值（侧栏已常驻），纯增复杂度。

## 影响

- 新分组 id（worker/cms/channel/system）需要配套图标：`web/admin/public/icons/{worker,cms,channel,system}.svg` 新增，风格与既有描边图标一致。
- i18n 三语 `menu.groups` 增删同步（删 base/alarm，增四组，boss/aaa 改名），keys.test 键集一致性通过。
- 自定义角色走 `menu:<key>` 权限码过滤，不受分组调整影响。
