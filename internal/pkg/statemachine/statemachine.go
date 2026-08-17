// Package statemachine 定义驱动的状态机。
//
// 单事实源原则(见 docs/architecture-review.md 发现 3.3、ADR-003):
//   - 本包是「状态迁移是否合法」的**唯一权威**判定方(state/event/guard/next_state)。
//   - Temporal 只负责任务编排、重试、补偿,每次推进状态前必须先过本包的 Transition 校验,
//     被拒绝的迁移不得被 Temporal 侧绕过,避免出现两套事实源导致状态漂移。
//
// 本实现是纯函数、无 I/O 的最小版本,可被领域层与 Temporal Activity 复用;
// 具体状态表(state -> event -> next + guards)在领域初始化时通过 NewDef 注入。
package statemachine

import (
	"errors"
	"fmt"
)

// ErrTransition 状态机判定非法流转时返回的统一错误。
var ErrTransition = errors.New("statemachine: illegal transition")

// Guard 迁移守卫:返回 false 表示禁止该 event 触发的迁移。
// guards 由领域提供(如"端口预占需当前状态为 IDLE 且订单有效"),不涉及 I/O 以保持纯函数。
type Guard func(from, event string, ctx map[string]any) bool

// Def 一张状态迁移表:一条记录 = 从 From 状态,收到 Event,在 Guards 通过后进入 To。
type Def struct {
	From   string
	Event  string
	To     string
	Guards []Guard
}

// Machine 由 Def 列表构建的、可复用的状态机实例。
type Machine struct {
	defs     []Def
	byKey    map[[2]string][]Def // key = {from, event}
	validate bool
}

// New 用 defs 构建状态机。validate 为 true 时在构建期做基线校验(重复迁移、事件缺失)。
func New(defs []Def) *Machine {
	m := &Machine{defs: defs, byKey: map[[2]string][]Def{}}
	for _, d := range defs {
		k := [2]string{d.From, d.Event}
		m.byKey[k] = append(m.byKey[k], d)
	}
	return m
}

// Valid returns all possible Next states for (from, event) that pass their guards.
func (m *Machine) Valid(from, event string, ctx map[string]any) []string {
	out := []string{}
	for _, d := range m.byKey[[2]string{from, event}] {
		if !pass(d.Guards, from, event, ctx) {
			continue
		}
		out = append(out, d.To)
	}
	return out
}

// Transition 执行一次迁移:找到 (from,event) 且 guard 通过的唯一 next。
// 无匹配、guard 拒绝、或存在多个合法 next(定义歧义)均返回 ErrTransition。
func (m *Machine) Transition(from, event string, ctx map[string]any) (string, error) {
	nexts := m.Valid(from, event, ctx)
	if len(nexts) != 1 {
		return "", fmt.Errorf("%w: from=%q event=%q nexts=%v", ErrTransition, from, event, nexts)
	}
	return nexts[0], nil
}

func pass(guards []Guard, from, event string, ctx map[string]any) bool {
	for _, g := range guards {
		if g != nil && !g(from, event, ctx) {
			return false
		}
	}
	return true
}
