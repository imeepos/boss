# SLO 基线快照(102,2026-08-26 采集)

> 采集命令:`scripts/ops/slo-collect.sh`(DB 侧 S4/S5/S7);HTTP 侧 S1/S2/S3 由 `/metrics` 常驻采集,见 slo.md。
> 本快照是首期基线,不作为达标判定依据(环境含大量测试数据);发布 1.0 后按周滚动判定。

## 采集结果(原始 JSON)

```json
{
  "s4_orders": { "window_7d_created": 151, "done": 43, "cancelled": 30 },
  "s5_push": { "by_status": { "LOG": 18 }, "window_7d": 18 },
  "s5_admin_todo": { "open": 1, "older_24h": 0 },
  "s7_reports": { "latest_daily_created": "2026-08-22T19:41:12Z",
                  "latest_daily_window": "2026-08-22T19:41:12Z",
                  "lag_seconds": 0, "on_time_cnt": 3 }
}
```

## 解读

| 指标 | 现状 | 说明 |
|:-----|:-----|:-----|
| S4 订单完成率 | 43/151(另有 30 取消) | 环境为联调库,含验收脚本造数与长期在途单,基线仅记录 |
| S5 推送 | 全部 LOG,无 SENT/FAILED | 102 未注册真实推送设备,真实送达率须在客户环境采集 |
| S5 待办时效 | open=1,无 >24h 积压 | 达标 |
| S7 报表时效 | 3/3 按时(lag 0s) | 窗口关闭即产出,达标 |

## 后续

- S5 真实送达率:1.0 验收时在客户设备注册后复采
- 本脚本纳入发布列车发布后验证清单,每列车跑一次留档
