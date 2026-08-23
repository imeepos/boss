package report

// 质量异常 → 补偿任务投递。
// 不直接 import metric 域，由调用方传入结构化数据（跨域禁实现依赖）。

import (
	"context"
	"fmt"
	"time"
)

// QualityViolationInput 质量违规输入（调用方从 metric.ScanQuality 转换）。
type QualityViolationInput struct {
	RuleKey  string
	Severity string
	Scope    string
	Detail   string
	Observed float64
}

// SubmitQualityViolations 将质量违规批量投递为补偿任务。
// 已存在的 ruleKey+OPEN 任务不重复创建（幂等）。
func (s *CompTaskService) SubmitQualityViolations(ctx context.Context, violations []QualityViolationInput, assigneeName string) (int, error) {
	if len(violations) == 0 {
		return 0, nil
	}
	// 加载已有 OPEN 任务，按 bizId 去重
	existing, _, err := s.St.ListCompTasks(ctx, CompTaskFilter{
		BizType: "metric_quality",
		Status:  TaskStatusOpen,
		Limit:   1000,
	})
	if err != nil {
		return 0, fmt.Errorf("metric: list existing quality tasks: %w", err)
	}
	openByRule := make(map[string]bool, len(existing))
	for _, t := range existing {
		openByRule[t.BizID] = true
	}

	now := time.Now()
	sla := now.Add(4 * time.Hour) // 4 小时 SLA（与四码冲突清零率一致）
	tasks := make([]CompTask, 0, len(violations))
	for _, v := range violations {
		if openByRule[v.RuleKey] {
			continue // 已有 OPEN 任务，跳过
		}
		tasks = append(tasks, CompTask{
			Source:        "QUALITY",
			BizType:       "metric_quality",
			BizID:         v.RuleKey,
			FailureReason: fmt.Sprintf("[%s] %s: %s (observed=%.4f)", v.Severity, v.Scope, v.Detail, v.Observed),
			Priority:      severityToPriority(v.Severity),
			Status:        TaskStatusOpen,
			AssigneeName:  assigneeName,
			SLADeadline:   &sla,
			MaxRetries:    3,
			AuditLog: []AuditEntry{{
				At:     now,
				Actor:  "system",
				Action: "create",
				Detail: fmt.Sprintf("quality scan: rule=%s severity=%s scope=%s", v.RuleKey, v.Severity, v.Scope),
			}},
		})
	}
	if len(tasks) == 0 {
		return 0, nil
	}
	if err := s.BatchCreate(ctx, tasks); err != nil {
		return 0, fmt.Errorf("metric: submit quality violations: %w", err)
	}
	return len(tasks), nil
}

func severityToPriority(severity string) string {
	switch severity {
	case "CRITICAL":
		return PriorityUrgent
	case "WARN":
		return PriorityHigh
	default:
		return PriorityNormal
	}
}
