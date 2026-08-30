# 环节7 下发模板解析规则：按套餐带宽匹配 + 法人默认回退（2026-08-30）

## 背景

`PreConfigOLT`（订单环节7 预下发配置）原实现直接把 `product_offers.id` 填进
`provision_tasks.template_id`，即"套餐 ID 当模板 ID 用"。102 实测（2026-08-31 前后核查）：

- `provision_templates`（1~139）与 `product_offers`（101~178）ID 空间重叠；
- 全部真实订单任务 `template_id=101`（100M 套餐），恰好撞上同号模板行
  `TPL-TN-931104027000`(tnet)——外键通过、模板语义完全错位；
- 套餐 ID >139（无同号模板行）会触发外键违规，环节7 直接失败、收费接口报错。

带宽的真正生效点在认证链路：RADIUS 授权实时 JOIN `product_offers.bandwidth`
回 FramedPool 属性；下发链路此前从未承载带宽语义。

## 裁定

环节7 模板解析走 `provision.FindTemplateForOffer(offerID, legalEntityID)`：

1. **带宽匹配优先**：`provision_templates.content->>'bandwidth'` = `product_offers.bandwidth`
   且 `status='ENABLED'`、同 `legal_entity_id`，取最旧一条；
2. **回退默认**：带宽无命中时取该法人最旧 ENABLED 模板，打
   `[provision] TEMPLATE FALLBACK` 可 grep 留痕（运维可见错配）；
3. **显性失败**：法人无任何可用模板 → 报错不推进环节7，禁止静默降级。

配套：

- `CreateTask` 补真幂等（原注释声称幂等、实际撞 `task_no` 唯一索引必报错）：
  同 `task_no` 复用既有任务，FAILED 重置 PENDING 留 RETRY 痕；
- 102 运维基线：seed 五档带宽模板（TPL-FTTH-100M/200M/300M/500M/1000M），
  compose 增加 `provisioner` 常驻服务，宿主机 systemd `boss-oltsim.service`
  仿真 OLT（telnet 2323）承接下发，链路端到端可验证。

## 放弃了什么

- **`product_offers` 加 `template_id` 外键列**：显式但强耦合——每建套餐必须先配模板，
  运维成本前置；且跨域写路径（customer 域产品表持有 provision 域外键）违反域边界惯例。
  带宽匹配让"新增套餐零配置命中既有档位模板"。
- **`provision_tasks` 加 params/bandwidth 列透传限速值**：当前 telnet 协议
  `template=<id>` 由设备侧模板承载配置，BOSS 不下发原始限速值；接真实 OLT
  需要限速数值时属厂商 VSA 适配（RADIUS 侧同类问题），单独立项。
- **`qos_template_id` 环节6 落值**：`qos_templates` 无 content 结构、无运维数据，
  落值无消费闭环；RADIUS 侧带宽兜底已可用。待 QoS 数据模型立项后接线。

## 已知残留

- `orders` 不快照带宽：改 `product_offers.bandwidth` 即时影响存量用户下次认证
  （既可视为灵活性也可视为风险，契约 fields.md 后续补注记）；
- 历史 DONE 任务的 `template_id=101` 错位留痕不重放（审计事实不动）；
- 102 曾发现验收清理漏删 `provision_tasks`（订单删了任务残留 PENDING），
  已手工清理 14 条孤儿，巡检脚本覆盖待后续。
