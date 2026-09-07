package odn

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// joinProblems 门控缺失明细拼装(透出给管理员)。
func joinProblems(problems []string) string {
	return strings.Join(problems, "; ")
}

// CheckProjectPermits 开工许可门控数据面(F3):判定 + 过期自动回写 EXPIRED。
// 满足条件:每类(ROW/PECE)存在达标许可或 NA;APPROVED 且 valid_until 已过 → 自动过期。
func (s *PGStore) CheckProjectPermits(ctx context.Context, projectID int64) (*PermitGateReport, error) {
	permits, err := s.ListProjectPermits(ctx, projectID)
	if err != nil {
		return nil, err
	}
	rep := EvaluatePermitGate(permits, time.Now().Format("2006-01-02"))
	if len(rep.ExpiredIDs) > 0 {
		tag, err := s.db.Exec(ctx, "UPDATE odn_permits SET status=$2, updated_at=now() "+
			"WHERE id = ANY($1) AND kind=$3 AND status=$4", rep.ExpiredIDs, PRowExpired, PermitKindROW, PRowApproved)
		if err != nil {
			log.Printf("[odn-permit] AUTO EXPIRE FAILED proj=%d ids=%v: %v", projectID, rep.ExpiredIDs, err)
			return nil, fmt.Errorf("odn: permit auto expire: %w", err)
		}
		log.Printf("[odn-permit] AUTO EXPIRED proj=%d ids=%v rows=%d", projectID, rep.ExpiredIDs, tag.RowsAffected())
	}
	if !rep.OK {
		return &rep, fmt.Errorf("odn: project %d %s: %w", projectID, joinProblems(rep.Problems), ErrPermitRequired)
	}
	return &rep, nil
}
