package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// idOrNil 把 0 归一为 NULL(可空外键约定:0=空)。
func idOrNil(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// ListDispatchTickets 列出全部派单工单。
func (s *PGStore) ListDispatchTickets(ctx context.Context) ([]DispatchTicket, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, ticket_no, order_id, COALESCE(worker_id, 0), COALESCE(worker_name, ''),
		       COALESCE(group_id, 0), COALESCE(group_name, ''), COALESCE(region_id, 0), COALESCE(region_name, ''),
		       legal_entity_id, legal_entity_name, status
		FROM dispatch_tickets ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list dispatch tickets: %w", err)
	}
	defer rows.Close()
	out := make([]DispatchTicket, 0)
	for rows.Next() {
		var t DispatchTicket
		if err := rows.Scan(&t.TicketID, &t.TicketNo, &t.OrderID, &t.WorkerID, &t.WorkerName,
			&t.GroupID, &t.GroupName, &t.RegionID, &t.RegionName, &t.LegalEntityID, &t.LegalEntityName, &t.Status); err != nil {
			return nil, fmt.Errorf("order: scan dispatch ticket: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListTicketItems 列表读模型:派单工单联表订单/客户/地址(师傅端列表页)。
// 详情视图(api/openapi/worker/schemas.yaml::TicketDetail)在此基础上复用:
//   增列 c.phone、po.name、t.{splitter_port,pre_bind_tag,schedule_slot}、
//   LEFT JOIN complaints(cmp) 取 faultTypeLabel/reportedAt/slaLeftMinutes/remoteDiagnosis。
//   订单 DONE 时取 MAX(order_stages.finished_at) 作为 finishedAt,空=进行中。
//   SLA 剩余分钟由 Go 侧实时计算(见 computeSlaLeft)。
func (s *PGStore) ListTicketItems(ctx context.Context) ([]TicketItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT t.id, t.ticket_no, t.order_id, COALESCE(t.worker_id, 0), t.status,
		       COALESCE(o.status, ''), COALESCE(c.name, ''), COALESCE(c.phone, ''), COALESCE(po.name, ''),
		       COALESCE(a.name, ua.detail, ''), COALESCE(o.stage, 0),
		       COALESCE(TO_CHAR(MAX(os.finished_at), 'YYYY-MM-DD HH24:MI'), ''),
		       COALESCE(t.splitter_port, ''), COALESCE(t.pre_bind_tag, ''), COALESCE(t.schedule_slot, ''),
		       COALESCE(cmp.type, ''),
		       COALESCE(TO_CHAR(cmp.created_at, 'YYYY-MM-DD HH24:MI'), ''),
		       COALESCE(cmp.remote_diagnosis, ''),
		       COALESCE(TO_CHAR(cmp.sla_deadline, 'YYYY-MM-DD HH24:MI'), '')
		FROM dispatch_tickets t
		LEFT JOIN orders o ON t.order_id = o.id
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN product_offers po ON o.offer_id = po.id
		LEFT JOIN addresses a ON o.address_id = a.id
		LEFT JOIN user_addresses ua ON o.address_id = ua.id
		LEFT JOIN order_stages os ON os.order_id = o.id
		LEFT JOIN complaints cmp ON cmp.order_id = o.id
		GROUP BY t.id, o.status, c.name, c.phone, po.name, a.name, ua.detail, o.stage,
		         t.splitter_port, t.pre_bind_tag, t.schedule_slot,
		         cmp.type, cmp.created_at, cmp.remote_diagnosis, cmp.sla_deadline
		ORDER BY t.id`)
	if err != nil {
		return nil, fmt.Errorf("order: list ticket items: %w", err)
	}
	defer rows.Close()
	out := make([]TicketItem, 0)
	for rows.Next() {
		var it TicketItem
		if err := rows.Scan(&it.TicketID, &it.TicketNo, &it.OrderID, &it.WorkerID, &it.Status,
			&it.OrderStatus, &it.CustomerName, &it.CustomerPhone, &it.OfferName,
			&it.Address, &it.Stage, &it.FinishedAt,
			&it.SplitterPort, &it.PreBindTag, &it.ScheduleSlot,
			&it.ComplaintType, &it.ReportedAt, &it.RemoteDiagnosis, &it.SlaDeadline); err != nil {
			return nil, fmt.Errorf("order: scan ticket item: %w", err)
		}
		it.FaultTypeLabel = complaintTypeLabels[it.ComplaintType]
		if it.FaultTypeLabel == "" && it.ComplaintType != "" {
			it.FaultTypeLabel = "报障（待分类）"
		}
		it.SlaLeftMinutes = computeSlaLeft(it.ComplaintType, it.SlaDeadline)
		out = append(out, it)
	}
	return out, rows.Err()
}

// GetTicketItemByNo 按 ticketNo 寻址的详情读模型:同 ListTicketItems 联表语义 + 单行过滤。
// 仅命中返回;无行时返回 ErrOrderNotFound(对齐 GetDispatchTicketByNo 行为)。
func (s *PGStore) GetTicketItemByNo(ctx context.Context, ticketNo string) (*TicketItem, error) {
	var it TicketItem
	err := s.db.QueryRow(ctx, `
		SELECT t.id, t.ticket_no, t.order_id, COALESCE(t.worker_id, 0), t.status,
		       COALESCE(o.status, ''), COALESCE(c.name, ''), COALESCE(c.phone, ''), COALESCE(po.name, ''),
		       COALESCE(a.name, ua.detail, ''), COALESCE(o.stage, 0),
		       COALESCE(TO_CHAR(MAX(os.finished_at), 'YYYY-MM-DD HH24:MI'), ''),
		       COALESCE(t.splitter_port, ''), COALESCE(t.pre_bind_tag, ''), COALESCE(t.schedule_slot, ''),
		       COALESCE(cmp.type, ''),
		       COALESCE(TO_CHAR(cmp.created_at, 'YYYY-MM-DD HH24:MI'), ''),
		       COALESCE(cmp.remote_diagnosis, ''),
		       COALESCE(TO_CHAR(cmp.sla_deadline, 'YYYY-MM-DD HH24:MI'), '')
		FROM dispatch_tickets t
		LEFT JOIN orders o ON t.order_id = o.id
		LEFT JOIN customers c ON o.customer_id = c.id
		LEFT JOIN product_offers po ON o.offer_id = po.id
		LEFT JOIN addresses a ON o.address_id = a.id
		LEFT JOIN user_addresses ua ON o.address_id = ua.id
		LEFT JOIN order_stages os ON os.order_id = o.id
		LEFT JOIN complaints cmp ON cmp.order_id = o.id
		WHERE t.ticket_no = $1
		GROUP BY t.id, o.status, c.name, c.phone, po.name, a.name, ua.detail, o.stage,
		         t.splitter_port, t.pre_bind_tag, t.schedule_slot,
		         cmp.type, cmp.created_at, cmp.remote_diagnosis, cmp.sla_deadline`,
		ticketNo).
		Scan(&it.TicketID, &it.TicketNo, &it.OrderID, &it.WorkerID, &it.Status,
			&it.OrderStatus, &it.CustomerName, &it.CustomerPhone, &it.OfferName,
			&it.Address, &it.Stage, &it.FinishedAt,
			&it.SplitterPort, &it.PreBindTag, &it.ScheduleSlot,
			&it.ComplaintType, &it.ReportedAt, &it.RemoteDiagnosis, &it.SlaDeadline)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order: get ticket item by no: %w", err)
	}
	it.FaultTypeLabel = complaintTypeLabels[it.ComplaintType]
	if it.FaultTypeLabel == "" && it.ComplaintType != "" {
		it.FaultTypeLabel = "报障（待分类）"
	}
	it.SlaLeftMinutes = computeSlaLeft(it.ComplaintType, it.SlaDeadline)
	return &it, nil
}

// computeSlaLeft SLA 剩余分钟计算。
// 优先用 sla_deadline(派单时写入的绝对截止时间);未写入时按 complaints.type
// 的 SLA 时长 + reportedAt 兜底计算;无 complaints 行或未知类型返回 0。
func computeSlaLeft(complaintType, slaDeadline string) int {
	if complaintType == "" {
		return 0
	}
	now := time.Now()
	if slaDeadline != "" {
		if dl, err := time.ParseInLocation("2006-01-02 15:04", slaDeadline, time.Local); err == nil {
			left := int(dl.Sub(now).Minutes())
			if left < 0 {
				return 0
			}
			return left
		}
	}
	return 0
}

// CreateDispatchTicket 新建派单工单,返回自增 id。
// 校验 order_id 存在性,防止孤儿工单。
func (s *PGStore) CreateDispatchTicket(ctx context.Context, t DispatchTicket) (int64, error) {
	// 关联完整性校验
	if t.OrderID > 0 {
		ok, err := s.exists(ctx, "orders", t.OrderID, "")
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("order: order %d not found for dispatch ticket: %w", t.OrderID, ErrOrderNotFound)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO dispatch_tickets(ticket_no, order_id, worker_id, worker_name, group_id, group_name, region_id, region_name, legal_entity_id, legal_entity_name, status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		t.TicketNo, t.OrderID, idOrNil(t.WorkerID), t.WorkerName, idOrNil(t.GroupID), t.GroupName,
		idOrNil(t.RegionID), t.RegionName, t.LegalEntityID, t.LegalEntityName, t.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: create dispatch ticket: %w", err)
	}
	return id, nil
}

// ListComplaints 列出全部报障工单。
func (s *PGStore) ListComplaints(ctx context.Context) ([]Complaint, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, ticket_no, customer_id, COALESCE(order_id, 0), legal_entity_id, legal_entity_name,
		       type, status,
		       COALESCE(TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI'), ''),
		       COALESCE(remote_diagnosis, ''),
		       COALESCE(TO_CHAR(sla_deadline, 'YYYY-MM-DD HH24:MI'), '')
		FROM complaints ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list complaints: %w", err)
	}
	defer rows.Close()
	out := make([]Complaint, 0)
	for rows.Next() {
		var c Complaint
		if err := rows.Scan(&c.ID, &c.TicketNo, &c.CustomerID, &c.OrderID,
			&c.LegalEntityID, &c.LegalEntityName, &c.Type, &c.Status,
			&c.CreatedAt, &c.RemoteDiagnosis, &c.SlaDeadline); err != nil {
			return nil, fmt.Errorf("order: scan complaint: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateComplaint 新建报障工单,返回自增 id。
// 校验 customer_id 和 order_id(若非零)存在性,防止孤儿投诉。
// Caller 可传 RemoteDiagnosis 和 SlaDeadline(格式 YYYY-MM-DD HH24:MI);空值走 DEFAULT。
func (s *PGStore) CreateComplaint(ctx context.Context, c Complaint) (int64, error) {
	// 关联完整性校验
	if c.CustomerID > 0 {
		ok, err := s.exists(ctx, "customers", c.CustomerID, "")
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("order: customer %d: %w", c.CustomerID, ErrForeignKeyViolation)
		}
	}
	if c.OrderID > 0 {
		ok, err := s.exists(ctx, "orders", c.OrderID, "")
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("order: order %d: %w", c.OrderID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO complaints(ticket_no, customer_id, order_id, legal_entity_id, legal_entity_name,
		                       type, status, remote_diagnosis, sla_deadline)
		VALUES($1,$2,$3,$4,$5,$6,$7,
		       $8,
		       CASE WHEN $9 = '' THEN NULL ELSE TO_TIMESTAMP($9, 'YYYY-MM-DD HH24:MI') END)
		RETURNING id`,
		c.TicketNo, c.CustomerID, idOrNil(c.OrderID), c.LegalEntityID, c.LegalEntityName,
		c.Type, c.Status, c.RemoteDiagnosis, c.SlaDeadline).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: create complaint: %w", err)
	}
	return id, nil
}

// CloseComplaint 投诉办结(ticketNo 寻址,status→CLOSED);未命中返回 ErrOrderNotFound。
func (s *PGStore) CloseComplaint(ctx context.Context, ticketNo string) error {
	tag, err := s.db.Exec(ctx, `UPDATE complaints SET status='CLOSED' WHERE ticket_no=$1`, ticketNo)
	if err != nil {
		return fmt.Errorf("order: close complaint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// GetDispatchTicketByNo 按工单号寻址(扫码闭环入口:ticketNo → orderID)。
func (s *PGStore) GetDispatchTicketByNo(ctx context.Context, ticketNo string) (*DispatchTicket, error) {
	var t DispatchTicket
	err := s.db.QueryRow(ctx, `
		SELECT id, ticket_no, order_id, COALESCE(worker_id, 0), COALESCE(worker_name, ''),
		       COALESCE(group_id, 0), COALESCE(group_name, ''), COALESCE(region_id, 0), COALESCE(region_name, ''),
		       legal_entity_id, legal_entity_name, status
		FROM dispatch_tickets WHERE ticket_no = $1`, ticketNo).
		Scan(&t.TicketID, &t.TicketNo, &t.OrderID, &t.WorkerID, &t.WorkerName,
			&t.GroupID, &t.GroupName, &t.RegionID, &t.RegionName, &t.LegalEntityID, &t.LegalEntityName, &t.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order: get dispatch ticket: %w", err)
	}
	return &t, nil
}

// AssignDispatchTicket 指派师傅:回填 worker_id/worker_name + 可选 schedule_slot/pre_bind_tag。
// opt 非空时追加写入(空串不覆盖已有值);未命中返回 ErrOrderNotFound。
func (s *PGStore) AssignDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string, opt ...AssignOpt) error {
	var o AssignOpt
	if len(opt) > 0 {
		o = opt[0]
	}
	if o.ScheduleSlot != "" || o.PreBindTag != "" {
		tag, err := s.db.Exec(ctx, `
			UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3,
				schedule_slot = CASE WHEN $4 = '' THEN schedule_slot ELSE $4 END,
				pre_bind_tag  = CASE WHEN $5 = '' THEN pre_bind_tag  ELSE $5 END
			WHERE ticket_no=$1`,
			ticketNo, workerID, workerName, o.ScheduleSlot, o.PreBindTag)
		if err != nil {
			return fmt.Errorf("order: assign dispatch ticket: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrOrderNotFound
		}
		return nil
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3 WHERE ticket_no=$1`,
		ticketNo, workerID, workerName)
	if err != nil {
		return fmt.Errorf("order: assign dispatch ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// ClaimDispatchTicket 师傅领取:回填师傅并 PENDING→DOING;非待派返回 ErrOrderNotFound。
func (s *PGStore) ClaimDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3, status='DOING'
		WHERE ticket_no=$1 AND status='PENDING'`,
		ticketNo, workerID, workerName)
	if err != nil {
		return fmt.Errorf("order: claim dispatch ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// AssignPendingDispatchTicket 抢单:待派且未指派时抢占,同时 PENDING→DOING(先到先得)。
func (s *PGStore) AssignPendingDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string, opt ...AssignOpt) error {
	var o AssignOpt
	if len(opt) > 0 {
		o = opt[0]
	}
	if o.ScheduleSlot != "" || o.PreBindTag != "" {
		tag, err := s.db.Exec(ctx, `
			UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3, status='DOING',
				schedule_slot = CASE WHEN $4 = '' THEN schedule_slot ELSE $4 END,
				pre_bind_tag  = CASE WHEN $5 = '' THEN pre_bind_tag  ELSE $5 END
			WHERE ticket_no=$1 AND status='PENDING' AND COALESCE(worker_id, 0)=0`,
			ticketNo, workerID, workerName, o.ScheduleSlot, o.PreBindTag)
		if err != nil {
			return fmt.Errorf("order: assign pending ticket: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrOrderNotFound
		}
		return nil
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE dispatch_tickets SET worker_id=$2, worker_name=$3, status='DOING'
		WHERE ticket_no=$1 AND status='PENDING' AND COALESCE(worker_id, 0)=0`,
		ticketNo, workerID, workerName)
	if err != nil {
		return fmt.Errorf("order: assign pending ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// UpdateScheduleSlot 改约:更新预约时间段 + 重置报障 SLA 截止时间。
func (s *PGStore) UpdateScheduleSlot(ctx context.Context, ticketNo string, scheduleSlot string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE dispatch_tickets SET schedule_slot=$2 WHERE ticket_no=$1`, ticketNo, scheduleSlot)
	if err != nil {
		return fmt.Errorf("order: update schedule slot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// ListScanLogs 列出扫码绑定记录;orderID=0 返回全部,否则按订单过滤。
func (s *PGStore) ListScanLogs(ctx context.Context, orderID int64) ([]ScanLog, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, order_id, worker_id, worker_name, tag_id, result FROM scan_logs WHERE ($1 = 0 OR order_id = $1) ORDER BY id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("order: list scan logs: %w", err)
	}
	defer rows.Close()
	out := make([]ScanLog, 0)
	for rows.Next() {
		var l ScanLog
		if err := rows.Scan(&l.ID, &l.OrderID, &l.WorkerID, &l.WorkerName, &l.TagID, &l.Result); err != nil {
			return nil, fmt.Errorf("order: scan scan log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// AppendScanLog 追加扫码绑定记录,返回自增 id。
func (s *PGStore) AppendScanLog(ctx context.Context, l ScanLog) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO scan_logs(order_id, worker_id, worker_name, tag_id, result)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		l.OrderID, l.WorkerID, l.WorkerName, l.TagID, l.Result).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: append scan log: %w", err)
	}
	return id, nil
}
