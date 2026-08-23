package billing

import "context"

// CollectionTaskItem is the operator queue projection for AR collection work.
type CollectionTaskItem struct {
	ID           int64   `json:"id"`
	CustomerID   int64   `json:"customerId"`
	CustomerName string  `json:"customer"`
	TaskType     string  `json:"taskType"`
	Priority     string  `json:"priority"`
	Status       string  `json:"status"`
	DueAt        string  `json:"dueAt"`
	Amount       float64 `json:"amount"`
	Days         int32   `json:"days"`
	Note         string  `json:"note"`
}

// CollectionQueueService keeps queue operations rule-driven and human-controlled.
type CollectionQueueService interface {
	ListCollectionTasks(ctx context.Context, status string) ([]CollectionTaskItem, error)
	UpdateCollectionTask(ctx context.Context, id int64, status, outcome, note string) error
}
