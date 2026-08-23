package metric

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// MemoryStore 内存实现，便于离线测试与单机原型。
type MemoryStore struct {
	mu      sync.RWMutex
	catalog map[string]CatalogEntry
	rules   map[string]QualityRule
}

// NewMemoryStore 构造内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{catalog: map[string]CatalogEntry{}, rules: map[string]QualityRule{}}
}

func (s *MemoryStore) ListCatalog(_ context.Context, status string) ([]CatalogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]CatalogEntry, 0, len(s.catalog))
	for _, e := range s.catalog {
		if status != "" && e.Status != status {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (s *MemoryStore) GetCatalogByKey(_ context.Context, key string) (*CatalogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.catalog[key]; ok {
		copy := e
		return &copy, nil
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) UpsertCatalog(_ context.Context, e CatalogEntry) (*CatalogEntry, error) {
	if e.Key == "" || e.Name == "" {
		return nil, fmt.Errorf("memory: key and name required")
	}
	if e.Status == "" {
		e.Status = StatusDraft
	}
	if e.RefreshCadence == "" {
		e.RefreshCadence = CadenceDaily
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.catalog[e.Key]; ok {
		existing.Version++
		existing.Name = e.Name
		existing.Description = e.Description
		existing.Formula = e.Formula
		existing.Unit = e.Unit
		existing.Dimensions = e.Dimensions
		existing.RefreshCadence = e.RefreshCadence
		existing.Owner = e.Owner
		existing.Domain = e.Domain
		existing.Status = e.Status
		s.catalog[e.Key] = existing
		return &existing, nil
	}
	e.Version = 1
	s.catalog[e.Key] = e
	return &e, nil
}

func (s *MemoryStore) DeprecateCatalog(_ context.Context, key string) (*CatalogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.catalog[key]
	if !ok {
		return nil, ErrNotFound
	}
	e.Status = StatusDeprecated
	s.catalog[key] = e
	return &e, nil
}

func (s *MemoryStore) ListQualityRules(_ context.Context, enabledOnly bool) ([]QualityRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]QualityRule, 0, len(s.rules))
	for _, r := range s.rules {
		if enabledOnly && !r.Enabled {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RuleKey < out[j].RuleKey })
	return out, nil
}

func (s *MemoryStore) UpsertQualityRule(_ context.Context, r QualityRule) (*QualityRule, error) {
	if r.RuleKey == "" || r.Name == "" || r.Scope == "" {
		return nil, fmt.Errorf("memory: rule_key/name/scope required")
	}
	if r.Severity == "" {
		r.Severity = SeverityWarn
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rules := s.rules
	rules[r.RuleKey] = r
	return &r, nil
}

func (s *MemoryStore) DisableQualityRule(_ context.Context, ruleKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rules[ruleKey]
	if !ok {
		return ErrNotFound
	}
	r.Enabled = false
	s.rules[ruleKey] = r
	return nil
}

func (s *MemoryStore) ScanQuality(_ context.Context, scope string) ([]QualityViolation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]QualityViolation, 0)
	for _, r := range s.rules {
		if !r.Enabled {
			continue
		}
		if scope != "" && r.Scope != scope {
			continue
		}
		out = append(out, QualityViolation{RuleKey: r.RuleKey, Severity: SeverityInfo, Scope: r.Scope, Detail: "scan_pending"})
	}
	return out, nil
}