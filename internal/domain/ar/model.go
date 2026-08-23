package ar

const (
	Standard   = "STANDARD"
	Watch      = "WATCH"
	Restricted = "RESTRICTED"
	Suspended  = "SUSPENDED"
)

// AgingSnapshot is the daily, explainable AR aging distribution.
type AgingSnapshot struct {
	CustomerID  int64
	Current     float64
	Days1To30   float64
	Days31To60  float64
	Days61To90  float64
	Days90Plus  float64
	Total       float64
}

// CollectionTask is an operator-owned, rule-generated queue item.
type CollectionTask struct {
	ID         int64  `json:"id"`
	CustomerID int64  `json:"customerId"`
	TaskType   string `json:"taskType"`
	Priority   string `json:"priority"`
	Status     string `json:"status"`
}
