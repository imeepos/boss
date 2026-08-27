package customer

import (
	"context"
	"time"
)

// ProductOffer 公司级产品(产品目录)。
// 字段权威:docs/contract/fields.md §2.2。
type ProductOffer struct {
	ID            int64     `json:"id"`
	LegalEntityID int64     `json:"legalEntityId"`
	Name          string    `json:"name"`
	Bandwidth     string    `json:"bandwidth"`
	MonthlyFee    float64   `json:"monthlyFee"` // NUMERIC(10,2)
	Category      string    `json:"category"`   // broadband/fusion/addon(用户端契约枚举),空回退 broadband
	EffectiveAt   time.Time `json:"effectiveAt"`
	Status        string    `json:"status"` // DRAFT/PUBLISHED/OFFLINE
}

// RegionOffer 区域运营包(同一产品在不同区域的名称与价格覆盖)。
type RegionOffer struct {
	ID         int64   `json:"id"`
	OfferID    int64   `json:"offerId"`
	RegionPath string  `json:"regionPath"`
	Name       string  `json:"name"` // 空=回退公司名(DB 列可空)
	MonthlyFee float64 `json:"monthlyFee"`
	Reason     string  `json:"reason"` // 空=无
}

// ProductService 产品资费域服务口(阶段2,与客户档案同包)。
type ProductService interface {
	ListProducts(ctx context.Context, legalEntityID int64) ([]ProductOffer, error)
	CreateProduct(ctx context.Context, p ProductOffer) (int64, error)
	ListRegionOffers(ctx context.Context, offerID int64) ([]RegionOffer, error)
	CreateRegionOffer(ctx context.Context, r RegionOffer) (int64, error)
	// ChangeProductPrice 产品调价:更新月费并追加调价台账,返回台账 id;产品不存在返回 ErrProductNotFound。
	ChangeProductPrice(ctx context.Context, offerID int64, newFee float64, effectiveAt time.Time, reason string, operatorAccountID int64) (int64, error)
	// UpdateProduct 编辑产品基础信息(名称/带宽/分类);月费走调价、状态走 UpdateStatus,公司归属不可改(防区域包孤儿)。
	UpdateProduct(ctx context.Context, offerID int64, name, bandwidth, category string) error
	// UpdateProductStatus 上下架;发布时刷新生效时间(与调价同口径),产品不存在返回 ErrProductNotFound。
	UpdateProductStatus(ctx context.Context, offerID int64, status string) error
}
