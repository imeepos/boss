// Package metric 提供指标目录与数据质量规则的领域能力。
//
// 设计目标：
//   - 单一指标事实源：所有指标的 key、定义、公式、负责人、版本都来自 metric_catalog；
//   - 质量异常统一派单到 compensation_tasks（已合并 7917bc1），不再依赖人工排查；
//   - 域不横向 import：仅暴露 Service 接口，由 internal/app 装配。
package metric
