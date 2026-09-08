package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// 勘测任务流转存储(W7):详情/指派/取消/接单/回填;状态机见 survey.go。

// GetSurvey 任务详情 + 回填记录(近单优先)。
func (s *PGStore) GetSurvey(ctx context.Context, id int64) (*SurveyTask, []SurveyReport, error) {
	t := &SurveyTask{}
	if err := scanSurvey(s.db.QueryRow(ctx, "SELECT "+surveyCols+surveyFrom+" WHERE t.id=$1", id), t); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		log.Printf("[odn-survey] GET FAILED id=%d: %v", id, err)
		return nil, nil, fmt.Errorf("odn: survey get: %w", err)
	}
	reports, err := s.listSurveyReports(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return t, reports, nil
}

func (s *PGStore) listSurveyReports(ctx context.Context, taskID int64) ([]SurveyReport, error) {
	rows, err := s.db.Query(ctx, "SELECT "+reportCols+reportFrom+" WHERE r.task_id=$1 ORDER BY r.id DESC", taskID)
	if err != nil {
		log.Printf("[odn-survey] REPORTS READ FAILED task=%d: %v", taskID, err)
		return nil, fmt.Errorf("odn: survey reports: %w", err)
	}
	defer rows.Close()
	out := []SurveyReport{}
	for rows.Next() {
		var r SurveyReport
		if err := rows.Scan(&r.ID, &r.TaskID, &r.WorkerID, &r.WorkerName, &r.Lat, &r.Lng,
			&r.FacilityNote, &r.Suggestion, &r.PhotoIDs, &r.ClientMsgID, &r.ReportedAt); err != nil {
			return nil, fmt.Errorf("odn: survey report scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// surveyTransition 指派/取消/接单共用的状态守卫更新:仅当当前状态在 allowed 内且
// (可选)当前指派等于 expectWorker(-1=不校验)时生效;返回是否命中。
func (s *PGStore) surveyTransition(ctx context.Context, id int64, allowed []string, expectWorker int64, workerID int64) (bool, error) {
	tag, err := s.db.Exec(ctx, "UPDATE survey_tasks SET assigned_worker_id=$2, updated_at=now() "+
		"WHERE id=$1 AND status = ANY($3) AND ($4 = -1 OR COALESCE(assigned_worker_id,0)=$4)",
		id, nullWorker(workerID), allowed, expectWorker)
	if err != nil {
		log.Printf("[odn-survey] TRANSITION FAILED id=%d worker=%d: %v", id, workerID, err)
		return false, fmt.Errorf("odn: survey transition: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func nullWorker(id int64) any {
	if id > 0 {
		return id
	}
	return nil
}

// AssignSurvey 指派/改派:仅 PENDING/ACCEPTED 可指派(回填后锁定)。
func (s *PGStore) AssignSurvey(ctx context.Context, id, workerID int64) error {
	if err := s.ensureAssignableWorker(ctx, workerID); err != nil {
		return err
	}
	var status string
	err := s.db.QueryRow(ctx, "SELECT status FROM survey_tasks WHERE id=$1", id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		log.Printf("[odn-survey] ASSIGN READ FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: survey assign read: %w", err)
	}
	if status != SurveyPending && status != SurveyAccepted {
		return fmt.Errorf("odn: survey status=%s: %w", status, ErrSurveyState)
	}
	ok, err := s.surveyTransition(ctx, id, []string{SurveyPending, SurveyAccepted}, -1, workerID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("odn: survey %d state=%s: %w", id, status, ErrSurveyState)
	}
	return nil
}

// CancelSurvey 取消:仅 PENDING/ACCEPTED 可取消(BACKFILLED 留痕不删)。
func (s *PGStore) CancelSurvey(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, "UPDATE survey_tasks SET status=$2, updated_at=now() "+
		"WHERE id=$1 AND status IN ($3,$4)", id, SurveyCancelled, SurveyPending, SurveyAccepted)
	if err != nil {
		log.Printf("[odn-survey] CANCEL FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: survey cancel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var status string
		err := s.db.QueryRow(ctx, "SELECT status FROM survey_tasks WHERE id=$1", id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			log.Printf("[odn-survey] CANCEL READ FAILED id=%d: %v", id, err)
			return fmt.Errorf("odn: survey cancel read: %w", err)
		}
		return fmt.Errorf("odn: survey status=%s: %w", status, ErrSurveyState)
	}
	return nil
}

// AcceptSurvey 师傅接单:未指派单抢单(校验接单资格并占位),已指派单仅指派师傅可接;
// 命中即置 ACCEPTED(状态流转与占位同语句,防抢到位未流转的中间态)。
func (s *PGStore) AcceptSurvey(ctx context.Context, id, workerID int64) error {
	if err := s.ensureAssignableWorker(ctx, workerID); err != nil {
		return err
	}
	var assignee int64
	err := s.db.QueryRow(ctx, "SELECT COALESCE(assigned_worker_id,0) FROM survey_tasks WHERE id=$1 AND status=$2", id, SurveyPending).Scan(&assignee)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("odn: survey %d not pending: %w", id, ErrSurveyState)
	}
	if err != nil {
		log.Printf("[odn-survey] ACCEPT READ FAILED id=%d worker=%d: %v", id, workerID, err)
		return fmt.Errorf("odn: survey accept read: %w", err)
	}
	if assignee > 0 && assignee != workerID {
		return fmt.Errorf("odn: survey %d assigned to %d: %w", id, assignee, ErrSurveyNotAssignee)
	}
	tag, err := s.db.Exec(ctx, "UPDATE survey_tasks SET status=$2, assigned_worker_id=$3, updated_at=now() "+
		"WHERE id=$1 AND status=$4 AND COALESCE(assigned_worker_id,0)=$5",
		id, SurveyAccepted, workerID, SurveyPending, assignee)
	if err != nil {
		log.Printf("[odn-survey] ACCEPT UPDATE FAILED id=%d worker=%d: %v", id, workerID, err)
		return fmt.Errorf("odn: survey accept update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("odn: survey %d state changed: %w", id, ErrSurveyState)
	}
	return nil
}

// AddSurveyReport 勘测回填(append-only):须已接单且为指派师傅;(task_id,client_msg_id) 幂等,
// 重复回显 created=false 不重复落行;首次回填置 BACKFILLED,其后追加不再改状态。
func (s *PGStore) AddSurveyReport(ctx context.Context, r SurveyReport) (int64, bool, error) {
	if err := ValidateSurveyReport(r); err != nil {
		return 0, false, err
	}
	var status string
	var assignee int64
	err := s.db.QueryRow(ctx, "SELECT status, COALESCE(assigned_worker_id,0) FROM survey_tasks WHERE id=$1", r.TaskID).
		Scan(&status, &assignee)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, ErrNotFound
	}
	if err != nil {
		log.Printf("[odn-survey] REPORT READ FAILED task=%d: %v", r.TaskID, err)
		return 0, false, fmt.Errorf("odn: survey report read: %w", err)
	}
	if assignee != r.WorkerID {
		return 0, false, fmt.Errorf("odn: survey %d assigned to %d: %w", r.TaskID, assignee, ErrSurveyNotAssignee)
	}
	if status != SurveyAccepted && status != SurveyBackfilled {
		return 0, false, fmt.Errorf("odn: survey status=%s: %w", status, ErrSurveyState)
	}
	var id int64
	err = s.db.QueryRow(ctx, "INSERT INTO survey_task_reports "+
		"(task_id, worker_id, lat, lng, facility_note, suggestion, photo_ids, client_msg_id) "+
		"VALUES ($1,$2,NULLIF($3,0),NULLIF($4,0),$5,$6,$7,$8) ON CONFLICT (task_id, client_msg_id) "+
		"DO NOTHING RETURNING id",
		r.TaskID, r.WorkerID, r.Lat, r.Lng, r.FacilityNote, r.Suggestion, r.PhotoIDs, r.ClientMsgID).Scan(&id)
	if err == nil {
		if err := s.markBackfilled(ctx, r.TaskID); err != nil {
			return 0, false, err
		}
		return id, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("[odn-survey] REPORT INSERT FAILED task=%d worker=%d msg=%s: %v", r.TaskID, r.WorkerID, r.ClientMsgID, err)
		return 0, false, fmt.Errorf("odn: survey report insert: %w", err)
	}
	err = s.db.QueryRow(ctx, "SELECT id FROM survey_task_reports WHERE task_id=$1 AND client_msg_id=$2",
		r.TaskID, r.ClientMsgID).Scan(&id)
	if err != nil {
		log.Printf("[odn-survey] REPORT IDEMPOTENT READ FAILED task=%d msg=%s: %v", r.TaskID, r.ClientMsgID, err)
		return 0, false, fmt.Errorf("odn: survey report idempotent read: %w", err)
	}
	return id, false, nil
}

// markBackfilled 首次回填置 BACKFILLED(幂等:已回填跳过)。
func (s *PGStore) markBackfilled(ctx context.Context, taskID int64) error {
	if _, err := s.db.Exec(ctx, "UPDATE survey_tasks SET status=$2, updated_at=now() "+
		"WHERE id=$1 AND status=$3", taskID, SurveyBackfilled, SurveyAccepted); err != nil {
		log.Printf("[odn-survey] BACKFILL MARK FAILED task=%d: %v", taskID, err)
		return fmt.Errorf("odn: survey backfill mark: %w", err)
	}
	return nil
}
