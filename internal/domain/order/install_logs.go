package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// InstallLog 施工回单(迁移 000164;N:1 dispatch_tickets)。
// status: OPEN=已提交待签收 / COMPLETED=已签收 / REJECTED=拒签。
// 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md §决策 2。
type InstallLog struct {
	ID           int64      `json:"id"`
	TicketID     int64      `json:"ticketId"`
	OrderID      int64      `json:"orderId"`
	WorkerID     int64      `json:"workerId"`
	WorkerName   string     `json:"workerName"`
	Photos       []int64    `json:"photos"` // attachments.id 列表
	SignName     string     `json:"signName"`
	SignImageURL string     `json:"signImageUrl"`
	SignedAt     *time.Time `json:"signedAt,omitempty"` // NULL=未签收(OPEN 态),不落 0001-01-01 假时刻
	Note         string     `json:"note"`
	Status       string     `json:"status"` // OPEN/COMPLETED/REJECTED
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// ArriveInput 师傅到场打卡入参(WGS84 经纬度 + 时间戳)。
// lat/lng 范围 [-90,90]/[-180,180];空时记为 NULL(老版本师傅端兜底)。
type ArriveInput struct {
	Lat       *float64   `json:"lat"`
	Lng       *float64   `json:"lng"`
	Timestamp *time.Time `json:"timestamp"` // nil=now()
}

// SubmitInstallLog 创建一条施工回单。
// 校验:ticketID 必须存在、workerID 必须匹配(防越权)、同 ticket 已有 OPEN 行直接 422。
func (s *PGStore) SubmitInstallLog(ctx context.Context, l InstallLog) (int64, error) {
	if l.TicketID <= 0 || l.OrderID <= 0 || l.WorkerID <= 0 {
		return 0, fmt.Errorf("order: install_log ticket/order/worker required: %w", ErrInstallInput)
	}
	if l.Status == "" {
		l.Status = "OPEN"
	}
	if l.Photos == nil {
		l.Photos = []int64{}
	}

	var ticketWorker int64
	if err := s.db.QueryRow(ctx,
		`SELECT COALESCE(worker_id, 0) FROM dispatch_tickets WHERE id=$1`,
		l.TicketID).Scan(&ticketWorker); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrOrderNotFound
		}
		return 0, fmt.Errorf("order: lookup ticket worker: %w", err)
	}
	if ticketWorker != l.WorkerID {
		return 0, fmt.Errorf("order: install_log worker %d != ticket worker %d: %w",
			l.WorkerID, ticketWorker, ErrInstallInput)
	}

	photosJSON, err := photosToJSON(l.Photos)
	if err != nil {
		return 0, err
	}

	var id int64
	err = s.db.QueryRow(ctx,
		`INSERT INTO install_logs(ticket_id, order_id, worker_id, worker_name, photos,
		                          sign_name, sign_image_url, signed_at, note, status)
		 VALUES($1,$2,$3,$4,$5::jsonb,$6,$7,$8,$9,$10) RETURNING id`,
		l.TicketID, l.OrderID, l.WorkerID, l.WorkerName, photosJSON,
		l.SignName, l.SignImageURL, l.SignedAt, l.Note, l.Status).Scan(&id)
	if err != nil {
		// uq_install_logs_ticket_open 唯一约束:同 ticket 已有 OPEN 行 → 业务拒绝(非 500)。
		if isUniqueViolation(err, "uq_install_logs_ticket_open") {
			slog.WarnContext(ctx, "[order] INSTALL LOG DUPLICATE",
				"ticket_id", l.TicketID, "worker_id", l.WorkerID,
				"reason", "ticket already has OPEN install_log")
			return 0, fmt.Errorf("order: ticket %d already has OPEN install_log: %w",
				l.TicketID, ErrInstallInput)
		}
		return 0, fmt.Errorf("order: insert install_log: %w", err)
	}
	return id, nil
}

// MarkArrived 师傅到场打卡:单事务回填 dispatch_tickets.arrived_at/arrive_lat/arrive_lng;
// 已有 arrived_at 不覆盖(幂等,留第一次;后续走 /rearrive 显式补录,本期不实现)。
// 校验:仅 DOING 工单可打卡;worker_id 匹配;lat/lng 范围(底层 CHECK 已拦)。
func (s *PGStore) MarkArrived(ctx context.Context, ticketNo string, in ArriveInput, workerID int64) error {
	ts := time.Now()
	if in.Timestamp != nil {
		ts = *in.Timestamp
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE dispatch_tickets
		 SET arrived_at=$2, arrive_lat=$3, arrive_lng=$4
		 WHERE ticket_no=$1 AND status='DOING' AND worker_id=$5 AND arrived_at IS NULL`,
		ticketNo, ts, in.Lat, in.Lng, workerID)
	if err != nil {
		slog.ErrorContext(ctx, "[order] ARRIVE UPDATE FAILED",
			"ticket_no", ticketNo, "worker_id", workerID, "err", err)
		return fmt.Errorf("order: arrive update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("order: ticket %s not DOING or worker mismatch or already arrived: %w",
			ticketNo, ErrInstallInput)
	}
	slog.InfoContext(ctx, "[order] ARRIVE OK",
		"ticket_no", ticketNo, "worker_id", workerID, "lat", in.Lat, "lng", in.Lng)
	return nil
}

// ListInstallLogs 按 ticket 列出回单(ORDER BY id DESC)。
func (s *PGStore) ListInstallLogs(ctx context.Context, ticketID int64) ([]InstallLog, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, ticket_id, order_id, worker_id, worker_name, photos,
		        COALESCE(sign_name,''), COALESCE(sign_image_url,''), signed_at,
		        COALESCE(note,''), status, created_at, updated_at
		 FROM install_logs WHERE ticket_id=$1 ORDER BY id DESC`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("order: list install_logs: %w", err)
	}
	defer rows.Close()
	out := []InstallLog{}
	for rows.Next() {
		var l InstallLog
		var photosJSON []byte
		if err := rows.Scan(&l.ID, &l.TicketID, &l.OrderID, &l.WorkerID, &l.WorkerName, &photosJSON,
			&l.SignName, &l.SignImageURL, &l.SignedAt, &l.Note, &l.Status, &l.CreatedAt, &l.UpdatedAt); err != nil {
			// signed_at 可空(OPEN 态未签收):pgx 扫 **time.Time,NULL 行不再炸整张列表
			return nil, err
		}
		ids, _ := jsonToPhotos(photosJSON)
		l.Photos = ids
		out = append(out, l)
	}
	return out, rows.Err()
}
