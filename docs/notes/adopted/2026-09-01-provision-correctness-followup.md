# 开通正确性收口：改套餐 LO 对齐 + 下单资格预检（2026-09-01）

> 背景：offer-provision-binding（方案B）验证轮实测暴露——改套餐订单在环节 7 仍按 LO 账号上的
> 旧套餐解析下发模板（订单 300M 实际按 LO 旧 100M 下发）。根因：环节 6 `CreateUserProfile`
> 幂等复用已有 LO 时直接推进，不刷新 `offer_id`/`billing_mode`。

## 裁定一：LO 是"当前生效套餐"权威态，环节 6 对齐（TMF change order 语义）

- `aaa.PGStore.AlignLoAccountOffer(customerID, offerID, billingMode)`：值变化才 UPDATE，
  返回是否更新；RADIUS 授权实时 JOIN `lo_accounts`，对齐即下次认证生效新档（模板下发与限速同源修正）。
- `order.CreateUserProfile` 幂等复用分支：`existing.OfferID != 订单套餐` 时调用对齐并打
  `[order] LO OFFER REALIGN` 可 grep 留痕。
- 语义边界：`lo_accounts` 为客户 1:1 单值模型，"当前生效"取最后到达环节 6 的订单；
  多在途订单并发场景不在本裁定范围（现状模型即单值，不引入新表）。

## 裁定二：下单资格预检（TMF Product Offering Qualification）

（环节 6/7 失败提前到下单时——见同日提交，下单 `Submit` 在派生法人后、插单前调
`FindTemplateForOffer`，不可解析即拒单：不可开通 = 不可售，杜绝"钱付了卡环节 7"。）

## 放弃了什么

- **环节 7 改用订单 offer 解析**：能修模板选择但修不了 RADIUS 旧档（授权 JOIN LO），治标不治本。
- **激活（环节 10）时才刷新 LO**：下发（环节 7）在前，刷新在后则下发仍用旧套餐。
- **增值类套餐跳过环节 7**：无绑定套餐经下单预检自然拒单（不可售），无需引入类别分支；
  若未来 IPTV 等需要独立开通通道，按服务类型扩展 provisioning handler，不与本裁定冲突。

## 已知债务

- `qos_templates` 死线：`lo_accounts.qos_template_id` 恒 0 且无内容模型——接线需 QoS 数据模型立项，
  移除列属破坏性操作，均待裁定。
