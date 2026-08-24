package app

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ymm-001/boss/internal/domain/metric"
	"github.com/ymm-001/boss/internal/domain/report"
)

type ETLScanResult struct {
	Checked    int      `json:"checked"`
	Overdue    int      `json:"overdue"`
	Dispatched int      `json:"dispatched"`
	JobKeys    []string `json:"jobKeys"`
	Errors     []string `json:"errors,omitempty"`
}

type ETLScanner interface {
	Scan(context.Context) (ETLScanResult, error)
	Latest() ETLScanResult
}

type etlScanner struct {
	etl    metric.ETLService
	comp   *report.CompTaskService
	mu     sync.RWMutex
	latest ETLScanResult
}

func NewETLScanner(etl metric.ETLService, comp *report.CompTaskService) ETLScanner {
	return &etlScanner{etl: etl, comp: comp}
}

func (s *etlScanner) Scan(ctx context.Context) (ETLScanResult, error) {
	result := ETLScanResult{JobKeys: []string{}}
	if s.etl == nil {
		return result, nil
	}
	freshness, err := s.etl.ListFreshness(ctx)
	if err != nil {
		return result, fmt.Errorf("etl freshness scan: %w", err)
	}
	result.Checked = len(freshness)
	violations := make([]report.QualityViolationInput, 0)
	recovered := make([]string, 0)
	for _, item := range freshness {
		if item.Status != metric.Overdue {
			if item.Status == metric.Fresh {
				recovered = append(recovered, item.JobKey)
			}
			continue
		}
		result.Overdue++
		result.JobKeys = append(result.JobKeys, item.JobKey)
		violations = append(violations, report.QualityViolationInput{
			RuleKey:  "etl_freshness:" + item.JobKey,
			Severity: metric.SeverityCritical,
			Scope:    "etl:" + item.JobKey,
			Detail:   fmt.Sprintf("任务%s超过%d分钟未刷新", item.Name, item.LatenessThreshold),
			Observed: float64(item.LatenessThreshold),
		})
	}
	if s.comp == nil {
		if len(violations) > 0 || len(recovered) > 0 {
			result.Errors = append(result.Errors, "compensation service unavailable")
		}
	} else {
		if len(recovered) > 0 {
			if _, err := s.comp.CloseRecoveredETL(ctx, recovered, "etl-owner"); err != nil {
				return s.record(result, fmt.Errorf("close recovered tasks: %w", err))
			}
		}
		if len(violations) > 0 {
			dispatched, err := s.comp.SubmitQualityViolations(ctx, violations, "etl-owner")
			result.Dispatched = dispatched
			if err != nil {
				return s.record(result, fmt.Errorf("dispatch overdue tasks: %w", err))
			}
		}
	}
	return s.record(result, nil)
}

func (s *etlScanner) record(result ETLScanResult, err error) (ETLScanResult, error) {
	s.mu.Lock()
	s.latest = result
	s.mu.Unlock()
	if err != nil {
		log.Printf("etl overdue scan failed: %v", err)
	}
	return result, err
}

func (s *etlScanner) Latest() ETLScanResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest
}
