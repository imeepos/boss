package cs

import "context"

// KnowledgeArticle 客服知识库条目(cs_knowledge_articles)。
type KnowledgeArticle struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Status      string `json:"status"` // DRAFT/PUBLISHED/OFFLINE
	Version     int32  `json:"version"`
	OwnerID     int64  `json:"ownerId"` // 0=未指定
	PublishedAt string `json:"publishedAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// KnowledgeService 知识库域服务口(客服工作台支撑)。
type KnowledgeService interface {
	ListArticles(ctx context.Context) ([]KnowledgeArticle, error)
	GetArticle(ctx context.Context, id int64) (*KnowledgeArticle, error)
	CreateArticle(ctx context.Context, a KnowledgeArticle) (int64, error)
	UpdateArticle(ctx context.Context, a KnowledgeArticle) error
	DeleteArticle(ctx context.Context, id int64) error
}
