package order

import (
	"context"
	"fmt"
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
	seq     int64
	cust    CustomerLookup
	checker ResourceChecker
	logSeq  int64
}

// NewMemoryService 创建内存订单服务。
func NewMemoryService(cust CustomerLookup, checker ResourceChecker) *MemoryService {
	return &MemoryService{
		m:       make(map[int64]*Order),
		logs:    make(map[int64][]StageLog),
		cust:    cust,
		checker: checker,
	}
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
		LegalEntityID: req.LegalEntityID,
		RegionPath:    req.RegionPath,
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

// Reserve 端口预占(环节3):状态机 PENDING→RESERVED。
func (s *MemoryService) Reserve(ctx context.Context, orderID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.m[orderID]
	if !ok {
		return ErrOrderNotFound
	}
	next, err := transition(o.Status, "reserve")
	if err != nil {
		return err
	}
	o.Status = next
	s.appendLogLocked(orderID, 3, "DONE")
	return nil
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
