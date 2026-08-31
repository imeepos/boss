# 产品/套餐 ↔ 下发模板显式绑定（方案B，2026-09-01）

## 背景

`product_offers`（产品/套餐）与 `provision_templates`（下发模板）此前无任何绑定关系，
环节7 `PreConfigOLT` 只靠 `FindTemplateForOffer` 按**套餐带宽**撞模板，撞不中回退**法人最旧 ENABLED 模板**。
102 实测（SQL 模拟当前解析逻辑）暴露三类真实故障：

- **无带宽套餐开错模板**：`IPTV高清电视(115)`/`电视增值包(105)`/`云存储100G(116)` 带宽为空，
  全部回退到模板 id=1 `TPL-E2E-985000`（E2E 测试模板）——买了 IPTV 下发的是测试配置；
- **有套餐无模板 → 开通卡死**：法人9 有上架产品 `10G(181)`，但该法人 0 条下发模板，
  `TEMPLATE MISSING` 报错，环节7 永不推进，订单死在第6步；
- **同带宽不同业务撞模板**：`家庭宽带300M(106)` 与 `5G融合套餐-129档(108)` 同为 300M，
  都命中 `TPL-FTTH-300M`；库内 `TPL-TN-*`(tnet) 模板永远匹配不上。

根因：带宽匹配是"猜测"，只能覆盖"宽带且按档位出模板"一种理想情况。
下单（环节1）不校验、错误拖到环节7（配置下发）才暴露——客户钱付了、装维上门才发现开不通/开错。

## 裁定

**方案B：显式绑定 + 带宽兜底 + 显性失败**（`Amended` 2026-08-30-preconfig-template-resolution 的"不加外键"与"回退默认"两点）。

1. **数据模型**：新表 `offer_provision_bindings`（迁移 000172），`offer_id` UNIQUE → 套餐唯一绑一个模板；
   冗余 `legal_entity_id` 企业锚点；绑定归 provision 域所有（`product_offers` 仅软引用读，域边界不破）。
2. **解析顺序**（`FindTemplateForOffer`）：
   ① 显式绑定：`offer_provision_bindings` JOIN 模板，模板须 ENABLED；绑了禁用模板 → 显性报错不静默降级；
   ② 带宽兜底：同法人 ENABLED 模板 `content->>'bandwidth'` = 套餐带宽，**无带宽套餐不参与兜底**（须显式绑定）；
   ③ 都未命中 → `TEMPLATE UNRESOLVED` 显性报错，**禁止回退法人任意模板**（那是开错配置的根源）。
3. **admin**：产品页新增"下发模板"列 + 行操作"绑定模板"（绑定/改绑/解绑抽屉，只列同法人 ENABLED 模板）；
   模板页新增"绑定套餐"计数列。读写沿用 `menu:product` / `menu:product-write`。
4. **可追踪**：`provision_logs` 执行/失败/重试日志补记 `template_id`/`template_code`
   （此前恒为 0，下发了哪个模板不可排查）。

## 放弃了什么

- **纯带宽匹配 + 法人最旧模板回退**：猜测性匹配，无带宽套餐全落到测试模板、同带宽不同业务无法区分。
- **`product_offers.template_id` 外键列（方案A）**：跨域外键 + 建套餐必须先配模板，运维成本前置；
  绑定表把"显式关系"与"套餐主档"解耦，绑定是可选配置，宽带套餐可不绑走带宽兜底。
- **下单环节1 预检**：暂缓——环节7 显性失败已满足"不许静默开错"的硬约束，避免改热路径；
  后续若需要"下单前拦截"，在 `Submit` 注入 `ProvisionTemplateFinder` 即可，不改变本裁定。

## 配套

- 迁移 `000172_offer_provision_binding`（up/down）。
- 接口：`GET /products/{id}/provision-binding`、`PUT/DELETE /products/{id}/provision-binding`、`GET /provision-bindings`（列）。
- 已知残留：存量无带宽上架产品（IPTV/云存储等）未绑定前，新订单会在环节7 显性失败（预期行为，倒逼补绑）；
  历史 `provision_tasks` 错位模板不重放（审计事实不动）。
