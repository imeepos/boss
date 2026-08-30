# 2026-09-07 支付 method='card' 双语义与对账口径裁定

日期: 2026-09-07

## 决策

`payments.method='card'` 现阶段同时承载两个语义:柜面 POS 线下刷卡(000167 三要素
siteName/counterCode/operatorName 非空)与 Stripe 线上卡通道(三要素全空,经 webhook
按 payNo 落账)。资金对账以**柜面三要素非空**为判别式剔除柜面 card,不对账范围;
scripts/ops/stripe-recon.sh 按此口径执行并单列柜面条数。

## why

stripe-recon 首跑报"本地有渠道无流水"(PAY-20260829015320-06E1),查库裁议
(先查库再接口复核):审计留痕为 payment.record/cashier_wang/验收旗舰店,属柜面
代客收款选 card,非渠道漏单;Stripe 侧按金额 8800 分与时间窗双查均零 PI。
若不裁定口径,该假警报要么长期误报、要么诱使把柜面流水错送渠道对账。

## 放弃了什么(被否决项)

- 立即拆枚举(method 增 `pos_card`)或加 `source` 列:动 fields.md 枚举 + 迁移 +
  三端联动,收益不抵当日改动面;留作后续契约裁定(见遗留)。
- 以 payNo 前缀区分渠道来源:两路 payNo 同为 PAY- 前缀,无可靠区分度。
- 对账脚本改查 DB 直连而非 admin API:破坏 db-patrol-gate 建立的
  "脚本仅依赖 curl+python3、经 API 消费"惯例。

## 遗留

fields.md 应把 `method` 枚举的 card 双语义写清(或后续拆枚举/加 source 列),
消除下游每个消费者自行判别的需要。

## 关联

- adopted/2026-08-23-stripe-card-channel.md(卡通道与 payNo 幂等)
- 迁移 000167 柜面凭证三要素(siteName/counterCode/operatorName)与
  meeting-minutes/2026-08-28-柜面现金收款.md(method 管资金通道、三列管人员归因)
- scripts/ops/stripe-recon.sh(判别式落地处,commit b5fe3f22)
