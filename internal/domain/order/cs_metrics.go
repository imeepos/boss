package order

import "context"

type CSMetrics struct {
	OpenCount        int     `json:"openCount"`
	ProcessingCount  int     `json:"processingCount"`
	ClosedCount      int     `json:"closedCount"`
	SLABreachedOpen  int     `json:"slaBreachedOpen"`
	SLAOnTimeClosed  int     `json:"slaOnTimeClosed"`
	SLAOverdueClosed int     `json:"slaOverdueClosed"`
	AvgCloseHours    float64 `json:"avgCloseHours"`
}

type CSMetricsReader interface {
	CSMetrics(context.Context) (*CSMetrics, error)
}
