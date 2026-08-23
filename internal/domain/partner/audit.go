package partner

import "context"

// AuditReportRow 渠道审计报表行，来源为既有 audit_logs。
type AuditReportRow struct {
	Action string `json:"action"`
	Count  int64  `json:"count"`
	LastAt string `json:"lastAt,omitempty"`
}

type AuditReportService interface {
	ListAuditReport(ctx context.Context, accountID int64, action string) ([]AuditReportRow, error)
}
