package worker

// 装维队管理 PG 实现(TeamService,000141)。
// 实体权威:migrations/000016(队伍/师傅)+000016 之外由 000141 增补(软删/归属台账)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpdateGroup 改名 + 指定队长(leaderID=0 清空);队长必须为本队在职成员。
func (s *PGStore) UpdateGroup(ctx context.Context, groupID int64, name string, leaderID int64) (*Group, error) {
	leaderName := ""
	if leaderID > 0 {
		err := s.db.QueryRow(ctx,
			`SELECT name FROM workers WHERE id = $1 AND group_id = $2 AND status = 1`,
			leaderID, groupID).Scan(&leaderName)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLeaderNotMember
		}
		if err != nil {
			return nil, fmt.Errorf("worker: check leader %d: %w", leaderID, err)
		}
	}
	var g Group
	err := s.db.QueryRow(ctx, `
		UPDATE worker_groups SET name = $2, leader_id = $3, leader_name = $4
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, legal_entity_id, code, name, COALESCE(leader_id, 0), COALESCE(leader_name, '')`,
		groupID, name, idOrNil(leaderID), leaderName).
		Scan(&g.ID, &g.LegalEntityID, &g.Code, &g.Name, &g.LeaderID, &g.LeaderName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("worker: update group: %w", err)
	}
	return &g, nil
}

// SoftDeleteGroup 软删队伍;仍有在职成员时拒绝(FK 语义见 fields.md §7.3 铁律 1)。
func (s *PGStore) SoftDeleteGroup(ctx context.Context, groupID int64) error {
	var active int
	if err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM workers WHERE group_id = $1 AND status = 1`, groupID).
		Scan(&active); err != nil {
		return fmt.Errorf("worker: count members of group %d: %w", groupID, err)
	}
	if active > 0 {
		return ErrGroupNotEmpty
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE worker_groups SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, groupID)
	if err != nil {
		return fmt.Errorf("worker: soft delete group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TransferWorker 成员调队:单条 CTE 原子完成 改 group_id + 关旧台账行 + 开新台账行。
// 新行快照取目标队伍(group_name/legal_entity_id)与师傅当前 region(region_id/region_name)。
func (s *PGStore) TransferWorker(ctx context.Context, t Transfer) error {
	tag, err := s.db.Exec(ctx, `
		WITH target AS (
			SELECT g.id, g.name, g.legal_entity_id, COALESCE(le.name, '') AS legal_entity_name
			FROM worker_groups g LEFT JOIN legal_entities le ON le.id = g.legal_entity_id
			WHERE g.id = $2 AND g.deleted_at IS NULL
		), upd AS (
			UPDATE workers SET group_id = $2
			WHERE id = $1 AND status = 1 AND group_id <> $2
			RETURNING id, region_id
		), closed AS (
			UPDATE worker_group_memberships SET effective_to = now()
			WHERE worker_id = $1 AND effective_to IS NULL
		)
		INSERT INTO worker_group_memberships(
			worker_id, group_id, group_name, legal_entity_id, legal_entity_name,
			region_id, region_name, reason, operator_account_id, effective_from)
		SELECT $1, tg.id, tg.name, tg.legal_entity_id, tg.legal_entity_name,
		       u.region_id, COALESCE(r.name, ''), $3, $4, now()
		FROM target tg, upd u LEFT JOIN regions r ON r.id = u.region_id`,
		t.WorkerID, t.TargetGroupID, t.Reason, idOrNil(t.OperatorAccountID))
	if err != nil {
		return fmt.Errorf("worker: transfer worker: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// 目标队不存在(或已软删) / 师傅不存在或离职 / 已在目标队,逐一分辨给调用方可读错误。
		var ok bool
		if err := s.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM worker_groups WHERE id = $1 AND deleted_at IS NULL)`,
			t.TargetGroupID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("worker: target group %d: %w", t.TargetGroupID, ErrForeignKeyViolation)
		}
		w, err := s.GetWorker(ctx, t.WorkerID)
		if err != nil {
			return err
		}
		if w.Status != 1 {
			return fmt.Errorf("worker %d: %w", t.WorkerID, ErrNotFound)
		}
		return ErrSameGroup
	}
	return nil
}

// ListTeamPerformances 队成员月度绩效:workers LEFT JOIN worker_performances(按成员聚合)。
func (s *PGStore) ListTeamPerformances(ctx context.Context, groupID int64, period string) ([]TeamMemberPerf, error) {
	rows, err := s.db.Query(ctx, `
		SELECT w.id, w.staff_no, w.name, w.phone,
		       (COALESCE(g.leader_id, 0) = w.id) AS is_leader,
		       COALESCE(SUM(p.finished), 0)::INT,
		       COALESCE(ROUND(AVG(p.on_time_rate)), 0)::SMALLINT,
		       COALESCE(ROUND(AVG(p.score)), 1)::FLOAT8
		FROM workers w
		JOIN worker_groups g ON g.id = w.group_id
		LEFT JOIN worker_performances p ON p.worker_id = w.id AND ($2 = '' OR p.period = $2)
		WHERE w.group_id = $1 AND w.status = 1
		GROUP BY w.id, g.leader_id
		ORDER BY is_leader DESC, SUM(p.finished) DESC NULLS LAST, w.id`,
		groupID, period)
	if err != nil {
		return nil, fmt.Errorf("worker: list team performances: %w", err)
	}
	defer rows.Close()
	out := make([]TeamMemberPerf, 0)
	for rows.Next() {
		var m TeamMemberPerf
		if err := rows.Scan(&m.WorkerID, &m.StaffNo, &m.Name, &m.Phone, &m.IsLeader,
			&m.Finished, &m.OnTimeRate, &m.Score); err != nil {
			return nil, fmt.Errorf("worker: scan team performance: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GroupOfLeader 查队长所辖队伍(软删队伍不认);非队长返回 ErrNotFound。
func (s *PGStore) GroupOfLeader(ctx context.Context, workerID int64) (*Group, error) {
	g := &Group{}
	err := s.db.QueryRow(ctx, `
		SELECT g.id, g.legal_entity_id, g.code, g.name,
		       COALESCE(g.leader_id, 0), COALESCE(g.leader_name, ''),
		       (SELECT COUNT(*) FROM workers w WHERE w.group_id = g.id AND w.status = 1)
		FROM worker_groups g
		WHERE g.leader_id = $1 AND g.deleted_at IS NULL`, workerID).
		Scan(&g.ID, &g.LegalEntityID, &g.Code, &g.Name, &g.LeaderID, &g.LeaderName, &g.MemberCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("worker: group of leader: %w", err)
	}
	return g, nil
}
