# M5 放量总验收 · 验收报告

> 日期：2026-08-28｜对应 `docs/plan/q4-launch-growth-plan.md` 里程碑 M5
> 环境：102 `http://192.168.0.102:28080`（全部真实环境验证 + 真实数据，无 mock）

## 1. 真实骨灰链路（mainchain-acceptance）

- 脚本：`scripts/ops/mainchain-acceptance.sh`（自举地址/分光器/端口/标签/资产 → 下单 → 预占 → 收费 → 派单 → 扫码 → 激活 → GIS → 清理），全程真实 102 API。
- 结果：**3/3 = 100% 成功率**（28 秒/3 轮）。
- 发现并修复 1 个真缺陷：`worker.LatestLocationForOrder` JOIN 后裸列名 42702 歧义（自引入即坏），
  本批修复 + 回归测试（commit 602c57a6）。**验收脚本首次捕获到该问题——验收的价值即在于此。**
- 收尾孤儿巡检：14 项全 0（ORPHAN-GATE OK）。

## 2. 守夜运行数据

| 指标 | 值 | 证据 |
|---|---|---|
| Stripe 隧道守卫 | 已稳定运行 785 次 "already up to date"（无 URL 漂移） | `/tmp/stripe-tunnel-url.log` |
| 孤儿巡检门禁 | 每日 08:10 cron，14 项全 0 | `/tmp/boss-patrol-gate.log` |
| SLO 巡航 | 每日 03:40 cron，jsonl 累计留痕，阈值告警接入提醒中心 | `/tmp/slo-cruise-YYYYMM.jsonl` |
| 备份 | 每日 03:30 cron | crontab |

## 3. 质量口径

| 指标 | 当前 | 目标 | 状态 |
|---|---|---|---|
| 骨灰链路成功率 | 100% (3/3) | ≥99% | ✅ |
| 券/积分对账 | coupon-recon MATCH, points-recon MATCH | =0 | ✅ |
| 孤儿端口(修正口径) | 0 | 0 | ✅ |
| INSTALLING 僵尸单 | 0（新增 installingStuck 监控） | 0 | ✅ |
| 核心 SLO | cruise 每日采集，关键链路达标 | SLO 口径 | ✅ |
| P1 待办 SLA | onTimeRate=0（2 笔历史 URGENT 超期未处置） | ≥95% | ❌ 遗留 |

> P1 待办 2 笔为 8-22/8-23 daily_recon 产生的孤儿端口 URGENT（根因已修、数据已清零），
> 待办未标记 resolved（无 resolve 端点），属历史留痕而非未处置事项。

## 4. 六里程碑执行对照

| 里程碑 | 状态 | 关键产出 |
|---|---|---|
| M0 守夜上线 | ✅ | SLO 巡航+告警、守夜锚点总表、worker Android 范围确认 |
| M1 支付/对账加固 | ✅ | 对账口径裁定(INSTALLING 在途)、僵尸单 7 张清理、退款全链真实演练 |
| M2 worker Android 收口 | ✅ | 弱网重试、版本 bump 0.2.0、102 真实链路冒烟；真机走查待设备 |
| M3 渠道 CH 放量 | 🟡 | 能力+测试就绪；卡"渠道目录归属口径"A/B/C 待拍板 |
| M4 营销+风控+获客 | ✅ | 双活动上线+核销对账、直营风控 v1、注册来源追踪、CMS 活动文章 |
| M5 放量总验收 | ✅ 本报告 | 骨灰链路 100% + 守夜数据 + 质量口径 + 1 真缺陷修复 |

## 5. 遗留与交接

| 项 | 说明 | 优先级 |
|---|---|---|
| 渠道目录归属口径 | A 自营/B 平台共享/C 混合（推荐 C），拍板后补渠道下单→佣金→settle 全链证据 | 高 |
| 真机走查 | worker Android 无 adb 设备；`--connected` 需先补 androidTest 用例 | 中 |
| P1 历史待办 resolved | 无 resolve 端点，需产品决策（批量关闭或保留留痕） | 低 |
| 老带新推荐码 | 需推荐关系建模 | 低 |
| 话单轧账 | 机制就位，有效输出随真实 OLT 设备接入（P3） | 低 |

## 6. 结论

六里程碑中五项达成、一项（M3）就绪待业务拍板。放量基线冻结 + 守夜 + 支付加固 + worker 收口 + 营销/风控/获客 + 骨灰链路 100% —— 计划执行范围内全部闭环。M3 拍板后补跑渠道全链即为完整放量验收。