package adminapi

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

func aaaAdminPage(c *gin.Context) aaa.AdminPage {
	return aaa.AdminPage{
		Page: queryInt(c, "page", 1), PageSize: queryInt(c, "pageSize", 20),
		Keyword: c.Query("keyword"), Status: c.Query("status"), Loid: c.Query("loid"),
	}
}

func aaaScope(c *gin.Context, svc interface {
	GetDataScope(ctx context.Context, accountID int64) (user.DataScope, error)
}) (aaa.AdminScope, error) {
	ds, err := svc.GetDataScope(c.Request.Context(), httpx.ClaimsAccountID(c))
	if err != nil {
		return aaa.AdminScope{}, err
	}
	return aaa.AdminScope{LegalEntityID: ds.LegalEntityID, RegionScope: ds.RegionScope}, nil
}

func queryInt(c *gin.Context, key string, fallback int) int {
	v, err := strconv.Atoi(c.Query(key))
	if err != nil || v < 1 {
		return fallback
	}
	return v
}
