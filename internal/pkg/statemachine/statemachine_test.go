package statemachine

import (
	"errors"
	"testing"
)

// buildMachine 构建测试用状态机: IDLE -> (start/IDLE_ONLY guard) -> RUNNING -> stop -> IDLE。
func buildMachine() *Machine {
	return New([]Def{
		{From: "IDLE", Event: "start", To: "RUNNING"},
		{From: "IDLE", Event: "reserve", To: "RESERVED", Guards: []Guard{func(_, _ string, ctx map[string]any) bool {
			return ctx["valid"] == true
		}}},
		{From: "RUNNING", Event: "stop", To: "IDLE"},
		{From: "IDLE", Event: "ambiguous", To: "A"},
		{From: "IDLE", Event: "ambiguous", To: "B"},
		{From: "IDLE", Event: "nilguard", To: "C", Guards: []Guard{nil}},
	})
}

func TestTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		event   string
		ctx     map[string]any
		want    string
		wantErr bool
	}{
		{name: "合法迁移", from: "IDLE", event: "start", want: "RUNNING"},
		{name: "guard 通过", from: "IDLE", event: "reserve", ctx: map[string]any{"valid": true}, want: "RESERVED"},
		{name: "guard 拒绝", from: "IDLE", event: "reserve", wantErr: true},
		{name: "无匹配事件", from: "RUNNING", event: "start", wantErr: true},
		{name: "多合法 next 视为歧义", from: "IDLE", event: "ambiguous", wantErr: true},
		{name: "nil guard 不拒绝", from: "IDLE", event: "nilguard", want: "C"},
	}
	m := buildMachine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.Transition(tt.from, tt.event, tt.ctx)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Transition(%q,%q) err=nil, want ErrTransition", tt.from, tt.event)
				}
				if !errors.Is(err, ErrTransition) {
					t.Fatalf("Transition(%q,%q) err=%v, want ErrTransition", tt.from, tt.event, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Transition(%q,%q): %v", tt.from, tt.event, err)
			}
			if got != tt.want {
				t.Fatalf("Transition(%q,%q)=%q, want %q", tt.from, tt.event, got, tt.want)
			}
		})
	}
}

func TestValid(t *testing.T) {
	m := buildMachine()
	if got := m.Valid("IDLE", "ambiguous", nil); len(got) != 2 {
		t.Fatalf("Valid(ambiguous)=%v, want 2 nexts", got)
	}
	if got := m.Valid("IDLE", "reserve", nil); len(got) != 0 {
		t.Fatalf("Valid(guard拒绝)=%v, want empty", got)
	}
	if got := m.Valid("RUNNING", "stop", nil); len(got) != 1 || got[0] != "IDLE" {
		t.Fatalf("Valid(stop)=%v, want [IDLE]", got)
	}
	if got := m.Valid("UNKNOWN", "start", nil); got == nil || len(got) != 0 {
		t.Fatalf("Valid(未知状态)=%v, want empty non-nil", got)
	}
}
