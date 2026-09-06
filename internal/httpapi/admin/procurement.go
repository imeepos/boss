package adminapi

// 采购-库存域路由(admin 端)。
// 路由前缀: /api/admin/v1
//   - GET/POST   /procurement/suppliers                          供应商增查
//   - PUT        /procurement/suppliers/:id                      供应商编辑(P2-W2-T2,编码不可改)
//   - POST       /procurement/suppliers/:id/enable               启用供应商(P2-W2-T2,幂等)
//   - POST       /procurement/suppliers/:id/disable              禁用供应商
//   - GET/POST   /procurement/orders                            采购单增查
//   - GET/PUT    /procurement/orders/:id                        详情/草稿编辑(P2-W2-T2,仅 DRAFT)
//   - POST       /procurement/orders/:id/submit                  提交
//   - POST       /procurement/orders/:id/cancel                  取消
//   - POST       /procurement/receipts                          创建入库单
//   - POST       /procurement/receipts/:id/confirm               入库确认(同事务建批次+资产)
//   - POST       /procurement/receipts/:id/reject                入库驳回(P2-W2-T2,仅 DRAFT)
//   - GET        /procurement/receipts                          列入库单
//   - GET        /procurement/inventory                         库存查询

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

func registerProcurementRoutes(g *gin.RouterGroup, a *app.Application) {
	p := g.Group("/procurement", requirePerm(a.User, "menu:purchase"))
	p.GET("/suppliers", procurementListSuppliers(a))
	p.POST("/suppliers", procurementCreateSupplier(a))
	p.PUT("/suppliers/:id", procurementUpdateSupplier(a))
	p.POST("/suppliers/:id/enable", procurementEnableSupplier(a))
	p.POST("/suppliers/:id/disable", procurementDisableSupplier(a))

	p.GET("/orders", procurementListOrders(a))
	p.POST("/orders", procurementCreateOrder(a))
	p.GET("/orders/:id", procurementGetOrder(a))
	p.PUT("/orders/:id", procurementUpdateOrder(a))
	p.POST("/orders/:id/submit", procurementSubmitOrder(a))
	p.POST("/orders/:id/cancel", procurementCancelOrder(a))

	p.POST("/receipts", procurementCreateReceipt(a))
	p.POST("/receipts/:id/confirm", procurementConfirmReceipt(a))
	p.POST("/receipts/:id/reject", procurementRejectReceipt(a))
	p.GET("/receipts", procurementListReceipts(a))

	p.GET("/inventory", procurementListInventory(a))
}
