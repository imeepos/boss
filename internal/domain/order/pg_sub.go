package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ymm-001/boss/internal/pkg/clock"
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
		       legal_entity_id, legal_entity_name, status,
		       arrived_at, arrive_lat, arrive_lng
		FROM dispatch_tickets ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list dispatch tickets: %w", err)
	}
	defer rows.Close()
	out := make([]DispatchTicket, 0)
	for rows.Next() {
		var t DispatchTicket
		if err := rows.Scan(&t.TicketID, &t.TicketNo, &t.OrderID, &t.WorkerID, &t.WorkerName,
			&t.GroupID, &t.GroupName, &t.RegionID, &t.RegionName, &t.LegalEntityID, &t.LegalEntityName, &t.Status,
			&t.ArrivedAt, &t.ArriveLat, &t.ArriveLng); err != nil {
			return nil, fmt.Errorf("order: scan dispatch ticket: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListTicketItems 列表读模型:派单工单联表订单/客户/地址(师傅端列表页)。
// 详情视图(api/openapi/worker/schemas.yaml::TicketDetail)在此基础上复用:
//
//	增列 c.phone、po.name、t.{splitter_port,pre_bind_tag,schedule_slot}、
//	LEFT JOIN complaints(cmp) 取 faultTypeLabel/reportedAt/slaLeftMinutes/remoteDiagnosis。
//	订单 DONE 时取 MAX(order_stages.finished_at) 作为 finishedAt,空=进行中。
//	SLA 剩余分钟由 Go 侧实时计算(见 computeSlaLeft)。
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
		if dl, err := time.ParseInLocation("2006-01-02 15:04", slaDeadline, clock.Location()); err == nil {
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
// 关联完整性:order_id NOT NULL 外键,必须存在(order_id=0 直接拒,防孤儿工单)。
func (s *PGStore) CreateDispatchTicket(ctx context.Context, t DispatchTicket) (int64, error) {
	// 关联完整性校验
	if t.OrderID <= 0 {
		return 0, fmt.Errorf("order: order_id required: %w", ErrOrderNotFound)
	}
	ok, err := s.exists(ctx, "orders", t.OrderID, "")
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("order: order %d not found for dispatch ticket: %w", t.OrderID, ErrOrderNotFound)
	}

	var id int64
	err = s.db.QueryRow(ctx, `
		INSERT INTO dispatch_tickets(ticket_no, order_id, worker_id, worker_name, group_id, group_name, region_id, region_name, legal_entity_id, legal_entity_name, status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		t.TicketNo, t.OrderID, idOrNil(t.WorkerID), t.WorkerName, idOrNil(t.GroupID), t.GroupName,
		idOrNil(t.RegionID), t.RegionName, t.LegalEntityID, t.LegalEntityName, t.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: create dispatch ticket: %w", err)
	}
	return id, nil
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
// 关联完整性:order_id NOT NULL 外键,必须存在,缺失直接拒。
func (s *PGStore) AppendScanLog(ctx context.Context, l ScanLog) (int64, error) {
	if l.OrderID <= 0 {
		return 0, fmt.Errorf("order: order_id required: %w", ErrOrderNotFound)
	}
	ok, err := s.exists(ctx, "orders", l.OrderID, "")
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("order: order %d: %w", l.OrderID, ErrOrderNotFound)
	}
	var id int64
	err = s.db.QueryRow(ctx, `
		INSERT INTO scan_logs(order_id, worker_id, worker_name, tag_id, result)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		l.OrderID, l.WorkerID, l.WorkerName, l.TagID, l.Result).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("order: append scan log: %w", err)
	}
	return id, nil
}
