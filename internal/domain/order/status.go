package order

import "github.com/ymm-001/boss/internal/pkg/statemachine"

// 订单状态迁移最少集(后续环节按 terms.md 第 1 节逐步补 defs)。
// 单事实源(ADR-003):所有 Status 迁移必须过 statemachine.Transition,不许直接改字段。
var orderStatusDefs = []statemachine.Def{
	{From: "PENDING", Event: "reserve", To: "RESERVED"},
	{From: "RESERVED", Event: "release", To: "PENDING"},
}

// orderSM 订单状态机实例(仅本包内使用)。
var orderSM = statemachine.New(orderStatusDefs)

// transition 执行一次状态迁移,非法迁移报 ErrIllegalTransition。
func transition(from, event string) (string, error) {
	next, err := orderSM.Transition(from, event, nil)
	if err != nil {
		return "", ErrIllegalTransition
	}
	return next, nil
}
