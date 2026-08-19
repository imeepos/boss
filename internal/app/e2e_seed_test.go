package app_test

// e2e 种子构造器:集中各域种子对象,保持 e2e 主测试文件聚焦流程。

import (
	"fmt"
	"time"

	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/resource"
)

func resSeed(addressID int64, suffix string) resource.Resource {
	return resource.Resource{
		LegalEntityID: 1, Code: "SPL-E2E-" + suffix, Name: "E2E分光器",
		Type: "SPLITTER", AddressID: addressID, Status: "ONLINE",
	}
}

func portSeed(resID int64, s *e2eSeed, suffix string, seq int) resource.Port {
	return resource.Port{
		PortCode:   fmt.Sprintf("P-E2E-%s-%02d", suffix, seq),
		QuadCode:   fmt.Sprintf("Q-E2E-%s-%02d", suffix, seq),
		ResourceID: resID, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		AddressID: s.addressID, RegionID: s.regionID, RegionName: s.regionName,
		Status: "IDLE",
	}
}

func chanSeed(suffix string) order.Channel {
	return order.Channel{Code: "E2E-" + suffix, Name: "E2E渠道", Status: "ACTIVE"}
}

func offerSeed(suffix string) customer.ProductOffer {
	return customer.ProductOffer{
		LegalEntityID: 1, Name: "E2E套餐" + suffix, Bandwidth: "300M",
		MonthlyFee: 99.0, EffectiveAt: time.Now(), Status: "PUBLISHED",
	}
}

func custSeed(s *e2eSeed) customer.Customer {
	return customer.Customer{
		Name: "E2E客户", Phone: "09170000000", IdType: "身份证", IdNo: "E2E-ID",
		RealNameStatus: "VERIFIED", ServiceStatus: "ACTIVE",
		AddressID: s.addressID, LegalEntityID: 1,
		RegionID: s.regionID, RegionName: s.regionName,
	}
}
