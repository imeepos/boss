package order

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

// CustomerLookup 下单时校验客户存在的跨域依赖口(契约 CT-001 反方向读取)。
// 由 app 装配层注入 customer 域实现,order 域不 import customer 域实现。
type CustomerLookup interface {
	// Exists 客户是否存在;不存在时 Submit 拒绝建单。
	Exists(ctx context.Context, id int64) (bool, error)
}

// MemoryService 阶段5骨架参考实现:进程内 map,仅供单测与协议链路验证。
// 阶段5落地替换为 DB(repo)+Redis 锁(端口预占互斥,见技术栈 3.1)。
type MemoryService struct {
	mu      sync.RWMutex
	m       map[int64]*Order
	logs    map[int64][]StageLog
	ratings map[string]Rating
	seq     int64
	cust    CustomerLookup
	checker ResourceChecker
	own     OwnershipResolver
	logSeq  int64
}

// NewMemoryService 创建内存订单服务;own 可为 nil(退回请求直传归属)。
func NewMemoryService(cust CustomerLookup, checker ResourceChecker, own ...OwnershipResolver) *MemoryService {
	s := &MemoryService{
		m:       make(map[int64]*Order),
		logs:    make(map[int64][]StageLog),
		ratings: make(map[string]Rating),
		cust:    cust,
		checker: checker,
	}
	if len(own) > 0 {
		s.own = own[0]
	}
	return s
}

// Submit 下单(环节1):校验客户存在后建单,status=PENDING、stage=1,写环节日志。
func (s *MemoryService) Submit(ctx context.Context, req SubmitReq) (*Order, error) {
	if req.ChannelID == 0 {
		return nil, fmt.Errorf("order: channel_id required")
	}
	ok, err := s.cust.Exists(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("order: customer %d not found", req.CustomerID)
	}
	legalEntityID, regionPath := req.LegalEntityID, req.RegionPath
	if s.own != nil {
		own, err := s.own.Resolve(ctx, req.AddressID)
		if err != nil {
			return nil, err
		}
		if err := checkOwnershipConflict(req, own); err != nil {
			return nil, err
		}
		legalEntityID, regionPath = own.LegalEntityID, own.RegionPath
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	o := &Order{
		ID:            s.seq,
		OrderNo:       fmt.Sprintf("ORD-%s-%03d", time.Now().Format("20060102"), s.seq),
		CustomerID:    req.CustomerID,
		OfferID:       req.OfferID,
		AddressID:     req.AddressID,
		Stage:         1,
		Status:        "PENDING",
		ChannelID:     req.ChannelID,
		LegalEntityID: legalEntityID,
		RegionPath:    regionPath,
		CreatedAt:     time.Now(),
	}
	s.m[o.ID] = o
	s.appendLogLocked(o.ID, 1, "DOING")
	return cloneOrder(o), nil
}

// CheckResource 资源核查(环节2):调 ResourceChecker,成功后推进 stage=2。
func (s *MemoryService) CheckResource(ctx context.Context, orderID int64) error {
	s.mu.RLock()
	o, ok := s.m[orderID]
	s.mu.RUnlock()
	if !ok {
		return ErrOrderNotFound
	}

	avail, _, err := s.checker.Check(ctx, o.AddressID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	o = s.m[orderID]
	if o.Stage < 1 {
		return ErrIllegalTransition
	}
	if o.Stage >= 2 {
		return nil // 幂等:已核查过不再重复推进
	}
	o.Stage = 2
	if !avail {
		s.appendLogLocked(orderID, 2, "PENDING")
		return nil // 无资源不报错,订单停在 stage=2 等待(REQ-OSS-001 可选方案)
	}
	s.appendLogLocked(orderID, 2, "DONE")
	return nil
}

// Reserve 端口预占(环节3):经工作流推进 stage=3 且 status PENDING→RESERVED。
func (s *MemoryService) Reserve(ctx context.Context, orderID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.advanceLocked(orderID, "reservePort")
}

// Track 返回订单副本与环节日志副本。
func (s *MemoryService) Track(ctx context.Context, orderID int64) (*Order, []StageLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.m[orderID]
	if !ok {
		return nil, nil, ErrOrderNotFound
	}
	logs := make([]StageLog, len(s.logs[orderID]))
	copy(logs, s.logs[orderID])
	return cloneOrder(o), logs, nil
}

// List 订单列表(内存实现:无客户/产品/地址名,只回基础字段)。
func (s *MemoryService) List(ctx context.Context, q OrderQuery) ([]OrderListItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]OrderListItem, 0)
	for _, o := range s.m {
		if q.Status != "" && !slices.Contains(strings.Split(q.Status, ","), o.Status) {
			continue
		}
		if q.CustomerID != 0 && o.CustomerID != q.CustomerID {
			continue
		}
		if q.Keyword != "" && !strings.Contains(o.OrderNo, q.Keyword) {
			continue
		}
		out = append(out, OrderListItem{
			OrderNo: o.OrderNo, Stage: o.Stage, Status: o.Status,
			AddressID: o.AddressID, CreatedAt: o.CreatedAt,
		})
	}
	return out, nil
}

// GetByNo 按订单号查订单;未命中返回 ErrOrderNotFound。
func (s *MemoryService) GetByNo(ctx context.Context, orderNo string) (*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.m {
		if o.OrderNo == orderNo {
			return cloneOrder(o), nil
		}
	}
	return nil, ErrOrderNotFound
}

// advanceLocked 推进一个环节(调用方须持锁);逻辑与 PGStore.advance 一致。
func (s *MemoryService) advanceLocked(orderID int64, event string) error {
	step, ok := workflowByEvent[event]
	if !ok {
		return fmt.Errorf("order: unknown event %q", event)
	}
	o, ok := s.m[orderID]
	if !ok {
		return ErrOrderNotFound
	}
	if o.Stage != step.stage-1 {
		return ErrIllegalTransition
	}
	if step.statusEvent != "" {
		ns, err := transition(o.Status, step.statusEvent)
		if err != nil {
			return err
		}
		o.Status = ns
	}
	o.Stage = step.stage
	s.appendLogLocked(orderID, step.stage, "DONE")
	return nil
}

// 环节 4~12(与 PGStore 同名,经同一工作流表推进)。
func (s *MemoryService) ChargeContract(ctx context.Context, orderID int64) error {
	return s.adv(orderID, "chargeContract")
}
func (s *MemoryService) ApplyTag(ctx context.Context, orderID int64) error {
	return s.adv(orderID, "applyTag")
}
func (s *MemoryService) CreateUserProfile(ctx context.Context, orderID int64) error {
	return s.adv(orderID, "createUserProfile")
}
func (s *MemoryService) PreConfigOLT(ctx context.Context, orderID int64) error {
	return s.adv(orderID, "preConfigOLT")
}
func (s *MemoryService) DispatchOrder(ctx context.Context, orderID int64) error {
	return s.adv(orderID, "dispatchOrder")
}
func (s *MemoryService) ScanBind(ctx context.Context, orderID int64) error {
	return s.adv(orderID, "scanBind")
}
func (s *MemoryService) ActivateUser(ctx context.Context, orderID int64) error {
	return s.adv(orderID, "activateUser")
}
func (s *MemoryService) NotifyActivation(ctx context.Context, orderID int64) error {
	return s.adv(orderID, "notifyActivation")
}
func (s *MemoryService) UpdateMap(ctx context.Context, orderID int64) error {
	return s.adv(orderID, "updateMap")
}

// Cancel 取消订单;Release 端口释放(预占回滚)。
func (s *MemoryService) Cancel(ctx context.Context, orderID int64) error {
	return s.transitionStatusLocked(orderID, "cancel")
}
func (s *MemoryService) Release(ctx context.Context, orderID int64) error {
	return s.transitionStatusLocked(orderID, "release")
}

func (s *MemoryService) adv(orderID int64, event string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.advanceLocked(orderID, event)
}

func (s *MemoryService) transitionStatusLocked(orderID int64, event string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.m[orderID]
	if !ok {
		return ErrOrderNotFound
	}
	next, err := transition(o.Status, event)
	if err != nil {
		return err
	}
	o.Status = next
	return nil
}

// appendLogLocked 写环节日志(调用方须持锁)。
func (s *MemoryService) appendLogLocked(orderID int64, stage int8, result string) {
	s.logSeq++
	s.logs[orderID] = append(s.logs[orderID], StageLog{ID: s.logSeq, OrderID: orderID, Stage: stage, Result: result})
}

func cloneOrder(o *Order) *Order {
	if o == nil {
		return nil
	}
	cp := *o
	return &cp
}

// ChangeAddress 变更安装地址(内存实现)。
func (s *MemoryService) ChangeAddress(ctx context.Context, orderID, addressID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.m[orderID]
	if !ok {
		return ErrOrderNotFound
	}
	o.AddressID = addressID
	return nil
}

// SaveRating 落订单评价(内存实现:覆写同单号评价)。
func (s *MemoryService) SaveRating(ctx context.Context, r Rating) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ratings == nil {
		s.ratings = make(map[string]Rating)
	}
	s.ratings[r.OrderNo] = r
	return nil
}

// RatingExists 订单是否已评价(内存实现)。
func (s *MemoryService) RatingExists(ctx context.Context, orderNo string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.ratings[orderNo]
	return ok, nil
}
