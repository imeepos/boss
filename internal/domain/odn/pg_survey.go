package odn

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// 勘测任务存储(P-INFRA-1 W7,迁移 000223):创建/列表/详情/指派/取消/接单/回填。

const surveyCols = "t.id, t.task_no, t.title, t.description, t.prv_code, t.city_prefix, t.grid_code, " +
	"COALESCE(t.assigned_worker_id,0), COALESCE(w.name,''), t.status, " +
	"(SELECT count(*) FROM survey_task_reports r WHERE r.task_id=t.id) AS report_count, " +
	"COALESCE(t.created_by,0), to_char(t.created_at,'YYYY-MM-DD HH24:MI:SS'), to_char(t.updated_at,'YYYY-MM-DD HH24:MI:SS')"

const surveyFrom = " FROM survey_tasks t LEFT JOIN workers w ON w.id=t.assigned_worker_id"

const reportCols = "r.id, r.task_id, r.worker_id, COALESCE(w.name,''), COALESCE(r.lat,0), COALESCE(r.lng,0), " +
	"r.facility_note, r.suggestion, r.photo_ids, r.client_msg_id, to_char(r.reported_at,'YYYY-MM-DD HH24:MI:SS')"

const reportFrom = " FROM survey_task_reports r LEFT JOIN workers w ON w.id=r.worker_id"

func scanSurvey(row pgx.Row, s *SurveyTask) error {
	return row.Scan(&s.ID, &s.TaskNo, &s.Title, &s.Description, &s.PrvCode, &s.CityPrefix, &s.GridCode,
		&s.AssignedWorkerID, &s.WorkerName, &s.Status, &s.ReportCount, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
}

// nextSurveyNo 生成勘测任务号 SV-YYYYMMDD-NNNNN(口径同结算单号,时间戳后 5 位兜底)。
func nextSurveyNo() string {
	return fmt.Sprintf("SV-%s-%05d", time.Now().Format("20060102"), time.Now().UnixNano()%100000)
}

// ensureAssignableWorker 师傅须在职(workerAssignable:status=1 且 left_at 为空)。
func (s *PGStore) ensureAssignableWorker(ctx context.Context, workerID int64) error {
	var ok bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM workers WHERE id=$1 AND status=1 AND left_at IS NULL)",
		workerID).Scan(&ok)
	if err != nil {
		log.Printf("[odn-survey] WORKER READ FAILED worker=%d: %v", workerID, err)
		return fmt.Errorf("odn: survey worker read: %w", err)
	}
	if !ok {
		return fmt.Errorf("odn: worker %d not assignable: %w", workerID, ErrSurveyNotAssignee)
	}
	return nil
}

// CreateSurvey admin 创建勘测任务(可带指派师傅;0=未指派进抢单池)。
func (s *PGStore) CreateSurvey(ctx context.Context, in SurveyCreateInput) (*SurveyTask, error) {
	if err := ValidateSurveyCreate(in); err != nil {
		return nil, err
	}
	if in.AssignedWorkerID > 0 {
		if err := s.ensureAssignableWorker(ctx, in.AssignedWorkerID); err != nil {
			return nil, err
		}
	}
	no := nextSurveyNo()
	t := &SurveyTask{TaskNo: no, Title: in.Title, Description: in.Description, PrvCode: in.PrvCode,
		CityPrefix: in.CityPrefix, GridCode: in.GridCode, AssignedWorkerID: in.AssignedWorkerID,
		Status: SurveyPending, CreatedBy: in.CreatedBy}
	var id int64
	var assignee any
	if in.AssignedWorkerID > 0 {
		assignee = in.AssignedWorkerID
	}
	var createdBy any
	if in.CreatedBy > 0 {
		createdBy = in.CreatedBy
	}
	err := s.db.QueryRow(ctx, "INSERT INTO survey_tasks "+
		"(task_no, title, description, prv_code, city_prefix, grid_code, assigned_worker_id, created_by) "+
		"VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, to_char(created_at,'YYYY-MM-DD HH24:MI:SS')",
		no, in.Title, in.Description, in.PrvCode, in.CityPrefix, in.GridCode, assignee, createdBy).
		Scan(&id, &t.CreatedAt)
	if err != nil {
		log.Printf("[odn-survey] CREATE FAILED title=%s no=%s: %v", in.Title, no, err)
		return nil, fmt.Errorf("odn: survey create: %w", err)
	}
	t.ID = id
	t.UpdatedAt = t.CreatedAt
	return t, nil
}

// ListSurveys admin 列表(可选状态/指派师傅过滤,近单优先)。
func (s *PGStore) ListSurveys(ctx context.Context, status string, assigneeID int64, limit int) ([]SurveyTask, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	where := " WHERE 1=1"
	args := []any{}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND t.status=$%d", len(args))
	}
	if assigneeID > 0 {
		args = append(args, assigneeID)
		where += fmt.Sprintf(" AND t.assigned_worker_id=$%d", len(args))
	}
	args = append(args, limit)
	rows, err := s.db.Query(ctx, "SELECT "+surveyCols+surveyFrom+where+" ORDER BY t.id DESC LIMIT $"+fmt.Sprint(len(args)), args...)
	if err != nil {
		log.Printf("[odn-survey] LIST FAILED status=%s worker=%d: %v", status, assigneeID, err)
		return nil, fmt.Errorf("odn: survey list: %w", err)
	}
	defer rows.Close()
	out := []SurveyTask{}
	for rows.Next() {
		var t SurveyTask
		if err := scanSurvey(rows, &t); err != nil {
			return nil, fmt.Errorf("odn: survey scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListSurveysForWorker 师傅端可见集:指派给自己的 + 未指派抢单池(取消单不可见)。
func (s *PGStore) ListSurveysForWorker(ctx context.Context, workerID int64, status string, limit int) ([]SurveyTask, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	where := " WHERE (t.assigned_worker_id=$1 OR t.assigned_worker_id IS NULL) AND t.status<>'" + SurveyCancelled + "'"
	args := []any{workerID}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND t.status=$%d", len(args))
	}
	args = append(args, limit)
	rows, err := s.db.Query(ctx, "SELECT "+surveyCols+surveyFrom+where+" ORDER BY t.id DESC LIMIT $"+fmt.Sprint(len(args)), args...)
	if err != nil {
		log.Printf("[odn-survey] WORKER LIST FAILED worker=%d: %v", workerID, err)
		return nil, fmt.Errorf("odn: survey worker list: %w", err)
	}
	defer rows.Close()
	out := []SurveyTask{}
	for rows.Next() {
		var t SurveyTask
		if err := scanSurvey(rows, &t); err != nil {
			return nil, fmt.Errorf("odn: survey scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
