package billing

import "context"

type AgingBuckets struct {
	D0To15  int `json:"d0To15"`
	D16To30 int `json:"d16To30"`
	D31To60 int `json:"d31To60"`
	D61To90 int `json:"d61To90"`
	D90Plus int `json:"d90Plus"`
}
type ARMetrics struct {
	TotalAmount      float64      `json:"totalAmount"`
	CustomerCount    int          `json:"customerCount"`
	StoppedCount     int          `json:"stoppedCount"`
	OverdueBillCount int          `json:"overdueBillCount"`
	AgingBuckets     AgingBuckets `json:"agingBuckets"`
	LastRunAt        string       `json:"lastRunAt"`
}
type ARMetricsReader interface {
	ARMetrics(context.Context) (*ARMetrics, error)
}
