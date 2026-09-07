package odn

import (
	"errors"
	"testing"
)

// 预算与里程碑编辑窗口单测(W6/G2,000218):
// 清单编辑(预算/增删/改名/计划日)仅 PENDING;状态标记至 ACCEPTED 前。
func TestValidateMilestoneEdit(t *testing.T) {
	if err := ValidateMilestoneEdit(CPending); err != nil {
		t.Errorf("PENDING 应可编辑: %v", err)
	}
	for _, st := range []string{CBuilding, CAccepted} {
		if err := ValidateMilestoneEdit(st); !errors.Is(err, ErrBudgetLocked) {
			t.Errorf("%s 编辑应 ErrBudgetLocked: %v", st, err)
		}
	}
}

func TestValidateMilestoneMark(t *testing.T) {
	for _, st := range []string{CPending, CBuilding} {
		if err := ValidateMilestoneMark(st); err != nil {
			t.Errorf("%s 应可标记: %v", st, err)
		}
	}
	if err := ValidateMilestoneMark(CAccepted); !errors.Is(err, ErrMilestoneLocked) {
		t.Errorf("ACCEPTED 标记应 ErrMilestoneLocked: %v", err)
	}
}

func TestValidateMilestoneStatus(t *testing.T) {
	if err := ValidateMilestoneStatus(MDone); err != nil {
		t.Errorf("DONE 合法: %v", err)
	}
	if err := ValidateMilestoneStatus(MPending); err != nil {
		t.Errorf("PENDING 合法: %v", err)
	}
	if err := ValidateMilestoneStatus("DONE2"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("未知状态应 42200: %v", err)
	}
}
