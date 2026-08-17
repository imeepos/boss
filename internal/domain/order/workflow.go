package order

// 订单 12 环节工作流(terms.md §1):环节=正交于 status 的「进行到第几步」维度。
// 单事实源:环节顺序在此表唯一维护,任何推进都必须过 advance,不许绕过。

// stageStep 一个环节推进事件。
type stageStep struct {
	event string // terms.md §1 标识
	stage int8   // 推进到的环节序号(2~12;环节1=Submit 建单)
	// statusEvent 该环节触发的 status 迁移事件(经 orderSM 判定);空=不改状态。
	statusEvent string
}

// workflow 环节 2~12 的顺序表。
var workflow = []stageStep{
	{event: "checkResource", stage: 2},
	{event: "reservePort", stage: 3, statusEvent: "reserve"},
	{event: "chargeContract", stage: 4},
	{event: "applyTag", stage: 5},
	{event: "createUserProfile", stage: 6},
	{event: "preConfigOLT", stage: 7},
	{event: "dispatchOrder", stage: 8, statusEvent: "install"},
	{event: "scanBind", stage: 9},
	{event: "activateUser", stage: 10},
	{event: "notifyActivation", stage: 11, statusEvent: "done"},
	{event: "updateMap", stage: 12},
}

// workflowByEvent 事件 → 环节步定义。
var workflowByEvent = func() map[string]stageStep {
	m := make(map[string]stageStep, len(workflow))
	for _, s := range workflow {
		m[s.event] = s
	}
	return m
}()
