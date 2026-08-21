package portal

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// memoryStore 内存实现:供单测/本地联调替身,与 PG 实现同一契约。
// 生产装配一律走 NewPGStore,本实现不参与 Application 装配。
type memoryStore struct {
	mu       sync.Mutex
	sms      map[string]string // phone|scene → code(固定 123456)
	accounts map[string]*Account
	prefs    map[int64]*Prefs
	messages map[int64][]Message
	wallets  map[int64]float64
	autopay  map[int64]bool
	seq      map[string]int64
	msgSeq   int64
}

// NewMemory 构造内存门户状态存储(仅测试/本地联调)。
func NewMemory() Service {
	return &memoryStore{
		sms:      map[string]string{},
		accounts: map[string]*Account{},
		prefs:    map[int64]*Prefs{},
		messages: map[int64][]Message{},
		wallets:  map[int64]float64{},
		autopay:  map[int64]bool{},
		seq:      map[string]int64{},
	}
}

func (s *memoryStore) IssueSms(_ context.Context, phone, scene string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sms[phone+"|"+scene] = "123456"
	return nil
}

func (s *memoryStore) ConsumeSms(_ context.Context, phone, scene, code string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := phone + "|" + scene
	want, ok := s.sms[key]
	if !ok || code == "" || code != want {
		return false, nil
	}
	delete(s.sms, key)
	return true, nil
}

func (s *memoryStore) LatestSmsCode(_ context.Context, phone, scene string) (*SmsCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code, ok := s.sms[phone+"|"+scene]
	if !ok {
		return nil, nil
	}
	return &SmsCode{Phone: phone, Scene: scene, Code: code, IssuedAt: time.Now()}, nil
}

func (s *memoryStore) NextSyntheticCustomerID(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq["CUST"]++
	return syntheticID(int64(s.seq["CUST"])), nil
}

func (s *memoryStore) UpsertAccount(_ context.Context, phone, password string, customerID int64) (*Account, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a := &Account{Phone: phone, CustomerID: customerID, PasswordHash: string(hash), PasswordUpdatedAt: time.Now()}
	s.accounts[phone] = a
	return a, nil
}

func (s *memoryStore) AccountByPhone(_ context.Context, phone string) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.accounts[phone]
	if !ok {
		return nil, ErrNotFound
	}
	return a, nil
}

func (s *memoryStore) AccountByCustomer(_ context.Context, customerID int64) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.accounts {
		if a.CustomerID == customerID {
			return a, nil
		}
	}
	return nil, ErrNotFound
}

func (s *memoryStore) VerifyPassword(ctx context.Context, phone, password string) (bool, error) {
	a, err := s.AccountByPhone(ctx, phone)
	if err != nil {
		return false, nil
	}
	return bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) == nil, nil
}

func (s *memoryStore) RebindPhone(_ context.Context, customerID int64, newPhone string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for phone, a := range s.accounts {
		if a.CustomerID == customerID {
			delete(s.accounts, phone)
			a.Phone = newPhone
			s.accounts[newPhone] = a
			return nil
		}
	}
	return ErrNotFound
}

func (s *memoryStore) GetPrefs(_ context.Context, customerID int64) (*Prefs, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.prefs[customerID]
	if !ok {
		p = &Prefs{Notify: map[string]any{}, Language: ""}
		s.prefs[customerID] = p
	}
	return p, nil
}

func (s *memoryStore) SavePrefs(ctx context.Context, customerID int64, notify map[string]any, language string) error {
	p, err := s.GetPrefs(ctx, customerID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if notify != nil {
		p.Notify = notify
	}
	if language != "" {
		p.Language = language
	}
	return nil
}

func (s *memoryStore) Messages(_ context.Context, customerID int64) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Message(nil), s.messages[customerID]...), nil
}

func (s *memoryStore) PutMessage(_ context.Context, customerID int64, payload map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgSeq++
	s.messages[customerID] = append([]Message{{
		ID: s.msgSeq, CustomerID: customerID, Payload: payload, CreatedAt: time.Now(),
	}}, s.messages[customerID]...)
	return nil
}

func (s *memoryStore) MarkAllRead(_ context.Context, customerID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.messages[customerID] {
		s.messages[customerID][i].Read = true
	}
	return nil
}

func (s *memoryStore) MarkRead(_ context.Context, customerID int64, messageID string) (bool, error) {
	id, err := strconv.ParseInt(messageID, 10, 64)
	if err != nil {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.messages[customerID] {
		if s.messages[customerID][i].ID == id {
			s.messages[customerID][i].Read = true
			return true, nil
		}
	}
	return false, nil
}

func (s *memoryStore) HasUnread(_ context.Context, customerID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.messages[customerID] {
		if !m.Read {
			return true, nil
		}
	}
	return false, nil
}

func (s *memoryStore) Balance(_ context.Context, customerID int64) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wallets[customerID], nil
}

func (s *memoryStore) AdjustBalance(_ context.Context, customerID int64, delta float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wallets[customerID] += delta
	return nil
}

func (s *memoryStore) NextNo(_ context.Context, kind string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq[kind]++
	return fmt.Sprintf("%s-%d", kind, s.seq[kind]), nil
}

func (s *memoryStore) AutoPay(_ context.Context, customerID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.autopay[customerID], nil
}

func (s *memoryStore) SetAutoPay(_ context.Context, customerID int64, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autopay[customerID] = enabled
	return nil
}
