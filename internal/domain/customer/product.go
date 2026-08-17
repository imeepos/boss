package customer

import (
	"context"
	"time"
)

// ProductOffer 公司级产品(产品目录)。
// 字段权威:docs/contract/fields.md §2.2。
type ProductOffer struct {
	ID            int64
	LegalEntityID int64
	Name          string
	Bandwidth     string
	MonthlyFee    float64 // NUMERIC(10,2)
	EffectiveAt   time.Time
	Status        string // DRAFT/PUBLISHED/OFFLINE
}

// RegionOffer 区域运营包(同一产品在不同区域的名称与价格覆盖)。
type RegionOffer struct {
	ID         int64
	OfferID    int64
	RegionPath string
	Name       string // 空=回退公司名(DB 列可空)
	MonthlyFee float64
	Reason     string // 空=无
}

// ProductService 产品资费域服务口(阶段2,与客户档案同包)。
type ProductService interface {
	ListProducts(ctx context.Context, legalEntityID int64) ([]ProductOffer, error)
	CreateProduct(ctx context.Context, p ProductOffer) (int64, error)
	ListRegionOffers(ctx context.Context, offerID int64) ([]RegionOffer, error)
	CreateRegionOffer(ctx context.Context, r RegionOffer) (int64, error)
}
