package order

import "github.com/ymm-001/boss/internal/pkg/statemachine"

// 订单状态迁移表(terms.md §3):status 是「订单当前状态」的正交维度。
// 单事实源(ADR-003):所有 Status 迁移必须过 statemachine.Transition,不许直接改字段。
var orderStatusDefs = []statemachine.Def{
	{From: "PENDING", Event: "reserve", To: "RESERVED"},
	{From: "RESERVED", Event: "release", To: "PENDING"},
	{From: "RESERVED", Event: "install", To: "INSTALLING"},
	{From: "INSTALLING", Event: "done", To: "DONE"},
	// 取消:任一未完成状态可取消(terms.md §3 CANCELLED)。
	{From: "PENDING", Event: "cancel", To: "CANCELLED"},
	{From: "RESERVED", Event: "cancel", To: "CANCELLED"},
	{From: "INSTALLING", Event: "cancel", To: "CANCELLED"},
	// 逆向迁移(环节回退):仅 RollbackStage 使用,正向流程禁止。
	{From: "INSTALLING", Event: "undispatch", To: "RESERVED"},
	{From: "DONE", Event: "undone", To: "INSTALLING"},
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
