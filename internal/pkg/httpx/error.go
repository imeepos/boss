package httpx

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/backup"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/cms"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/domain/loy"
	"github.com/ymm-001/boss/internal/domain/metric"
	"github.com/ymm-001/boss/internal/domain/monthly"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/partner"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/procurement"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/sms"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// RespondErr 领域错误 → 统一错误码。未知错误一律 500。
func RespondErr(c *gin.Context, err error) {
	var assetRefErr *asset.ErrAssetReferenced
	var assetIdentityDup *asset.ErrAssetIdentityDuplicate
	switch {
	case errors.Is(err, user.ErrUnauthorized):
		Respond(c, apitypes.CodeUnauthorized, nil)
	case errors.Is(err, user.ErrRoleProtected),
		errors.Is(err, partner.ErrRegionOutsideEnterprise):
		Respond(c, apitypes.CodeForbidden, nil)
	case errors.Is(err, user.ErrUsernameTaken),
		errors.Is(err, user.ErrStaffNoTaken),
		errors.Is(err, user.ErrDuplicate),
		errors.Is(err, user.ErrConflict),
		errors.Is(err, customer.ErrDuplicate),
		errors.Is(err, geo.ErrDuplicate),
		errors.Is(err, billing.ErrDuplicateInvoice),
		errors.Is(err, billing.ErrPaymentNotRefundable),
		errors.Is(err, order.ErrChannelDuplicate),
		errors.Is(err, cms.ErrSlugTaken),
		errors.Is(err, cms.ErrCategoryTaken),
		errors.Is(err, cms.ErrCategoryInUse),
		errors.Is(err, asset.ErrBindingConflict),
		errors.Is(err, asset.ErrTagUnbound),
		errors.Is(err, asset.ErrAssetScrapped),
		errors.Is(err, asset.ErrModelExists),
		errors.Is(err, asset.ErrModelInactive),
		errors.Is(err, asset.ErrCodeDuplicate),
		errors.Is(err, asset.ErrTagDisabled),
		errors.Is(err, asset.ErrAssetNotInStock),
		errors.Is(err, asset.ErrAssignmentClosed),
		errors.Is(err, asset.ErrReplacementNotCancellable),
		errors.Is(err, worker.ErrDuplicate),
		errors.Is(err, resource.ErrDuplicate),
		errors.Is(err, aaa.ErrDuplicate),
		errors.Is(err, aaa.ErrOfferNotPublished):
		// 资产/标签双绑冲突:40900 + 透传 err.Error()(含具体资产/标签 id),
		// 调用方能区分"标签已绑"vs"资产已绑",与 40920 扫码不一致明确区分。
		Respond(c, apitypes.CodeConflict, gin.H{"reason": err.Error()})
	case errors.As(err, &assetRefErr):
		// 资产删除命中引用(P2-W1-T1):40900 + 全量阻断项清单(message 列明标签绑定/
		// 持有台账/换新单/盘点明细/四码关联,操作员按单消除)。
		Respond(c, apitypes.CodeConflict, gin.H{"reason": assetRefErr.Error()})
	case errors.As(err, &assetIdentityDup):
		// 身份列唯一冲突(P3-T2 迁移 000188):40900 + reason 含冲突字段名(sn/mac/loid),
		// 操作员能直接定位是哪个身份标识撞了全网唯一。
		Respond(c, apitypes.CodeConflict, gin.H{"reason": err.Error()})
	case errors.Is(err, asset.ErrInvalidMAC),
		errors.Is(err, asset.ErrInvalidEPC):
		// 身份/EPC 格式非法(P3-T2):42200 + 原因透传,操作员可见哪个字段不合规范。
		Respond(c, apitypes.CodeInvalidParam, gin.H{"reason": err.Error()})
	case errors.As(err, new(*asset.ErrTypeNotAllowed)):
		// 类型白名单外写入(P4-T2 类型归一):字面 HTTP 400 + 42200,与列表 sort
		// 白名单越界(adminapi respondBadRequest)同一线上形态;reason 透传合法集合,
		// 操作员/调用方能直接改用权威类型码。不走 Respond(恒 200),裁定口径是白名单外 400。
		c.JSON(http.StatusBadRequest, gin.H{
			"code": apitypes.CodeInvalidParam,
			"msg":  apitypes.CodeInvalidParam.Message(),
			"data": gin.H{"reason": err.Error()},
		})
	case errors.Is(err, asset.ErrInvalidSort):
		// 列表排序白名单越界(P3-T1):42200;正常路径由 adminapi 解析层 400 拦截,此处兜底。
		Respond(c, apitypes.CodeInvalidParam, gin.H{"reason": err.Error()})
	case errors.Is(err, provision.ErrBindingInvalid):
		// 绑定校验失败(跨法人/模板停用):42200 + 透传原因,管理员可见为什么绑不上。
		Respond(c, apitypes.CodeInvalidParam, gin.H{"reason": err.Error()})
	case errors.Is(err, asset.ErrScrapConfirmMismatch):
		// 报废三要素确认不符(P3-F 防绕过前端):42200 + 透传只指明要素的原因,
		// 不回显服务端现值(防状态探测),操作员按提示重新核对实物铭牌。
		Respond(c, apitypes.CodeInvalidParam, gin.H{"reason": err.Error()})
	case errors.Is(err, provision.ErrTemplateUnresolved),
		errors.Is(err, provision.ErrBoundTemplateDisabled):
		// 环节7 模板不可解析(配置缺失/绑定模板停用):40900 + 透传原因,运营可见为什么开不了。
		Respond(c, apitypes.CodeConflict, gin.H{"reason": err.Error()})
	case errors.Is(err, procurement.ErrStateConflict):
		// 采购域状态冲突(P2-W2-T2 草稿编辑/入库驳回非 DRAFT):40900 + 透传当前状态。
		Respond(c, apitypes.CodeConflict, gin.H{"reason": err.Error()})
	case errors.Is(err, user.ErrInvalidInput),
		errors.Is(err, user.ErrRoleNotFound),
		errors.Is(err, user.ErrFKViolation),
		errors.Is(err, customer.ErrForeignKeyViolation),
		errors.Is(err, customer.ErrInvalidProductStatus),
		errors.Is(err, quadlink.ErrForeignKeyViolation),
		errors.Is(err, worker.ErrForeignKeyViolation),
		errors.Is(err, worker.ErrInvalidPassword),
		errors.Is(err, provision.ErrForeignKeyViolation),
		errors.Is(err, asset.ErrForeignKeyViolation),
		errors.Is(err, resource.ErrForeignKeyViolation),
		errors.Is(err, aaa.ErrForeignKeyViolation),
		errors.Is(err, asset.ErrBatchNotEditable),
		errors.Is(err, procurement.ErrForeignKey),
		errors.Is(err, procurement.ErrInvalidInput),
		errors.Is(err, procurement.ErrInvalidTransition),
		errors.Is(err, order.ErrInstallInput),
		errors.Is(err, backup.ErrInvalidInput),
		errors.Is(err, order.ErrInvalidInput),
		errors.Is(err, billing.ErrInvalidMethod),
		errors.Is(err, billing.ErrForeignKeyViolation),
		errors.Is(err, ErrGeoInvalidParam),
		errors.Is(err, cms.ErrInvalidPost),
		errors.Is(err, cms.ErrInvalidCategory),
		errors.Is(err, monthly.ErrInvalidInput):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, user.ErrNotFound),
		errors.Is(err, aaa.ErrNotFound),
		errors.Is(err, resource.ErrNotFound),
		errors.Is(err, aaa.ErrSessionNotFound),
		errors.Is(err, asset.ErrNotFound),
		errors.Is(err, procurement.ErrNotFound),
		errors.Is(err, customer.ErrCustomerNotFound),
		errors.Is(err, customer.ErrProductNotFound),
		errors.Is(err, order.ErrOrderNotFound),
		errors.Is(err, billing.ErrNotFound),
		errors.Is(err, geo.ErrNotFound),
		errors.Is(err, provision.ErrTaskNotFound),
		errors.Is(err, provision.ErrLogNotFound),
		errors.Is(err, customer.ErrRegistrationNotFound),
		errors.Is(err, customer.ErrRealNameNotFound),
		errors.Is(err, worker.ErrNotFound),
		errors.Is(err, worker.ErrRegistrationNotFound),
		errors.Is(err, worker.ErrRealNameNotFound),
		errors.Is(err, userdata.ErrNotFound),
		errors.Is(err, userdata.ErrPlanNotFound),
		errors.Is(err, partner.ErrApplicationNotFound),
		errors.Is(err, partner.ErrStaffNotFound),
		errors.Is(err, portal.ErrNotFound),
		errors.Is(err, backup.ErrNotFound),
		errors.Is(err, metric.ErrNotFound),
		errors.Is(err, metric.ErrETLNotFound),
		errors.Is(err, cms.ErrPostNotFound),
		errors.Is(err, cms.ErrCategoryNotFound),
		errors.Is(err, monthly.ErrUnknownTable),
		errors.Is(err, monthly.ErrNotFound):
		Respond(c, apitypes.CodeNotFound, nil)
	case errors.Is(err, backup.ErrBusy):
		Respond(c, apitypes.CodeResourceBusy, nil)
	// 装维队管理(000141):队非空软删/同队调队 → 冲突;队长非本队成员 → 参数非法。
	// 盘点差异(S10):存在未处置差异禁止关单 / 任务或明细状态不允许该操作 → 冲突。
	case errors.Is(err, worker.ErrGroupNotEmpty),
		errors.Is(err, worker.ErrSameGroup),
		errors.Is(err, asset.ErrDiffPending),
		errors.Is(err, asset.ErrStocktakeState):
		Respond(c, apitypes.CodeConflict, nil)
	case errors.Is(err, worker.ErrLeaderNotMember):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, odn.ErrDuplicate):
		Respond(c, apitypes.CodeConflict, nil)
	case errors.Is(err, odn.ErrInvalidLifecycle),
		errors.Is(err, odn.ErrInvalidPortState),
		errors.Is(err, odn.ErrInvalidProjStatus),
		errors.Is(err, odn.ErrItemLocked),
		errors.Is(err, odn.ErrNoFreePort):
		// 状态冲突(生命周期/端口/施工单):40900,管理员可见转移被拒。
		Respond(c, apitypes.CodeConflict, nil)
	case errors.Is(err, odn.ErrNoCoverageDevice):
		// 覆盖未挂设备(下单门控判据):40400,前端提示先补覆盖关联。
		Respond(c, apitypes.CodeNotFound, nil)
	case errors.Is(err, odn.ErrPortNotInService):
		// 绑定要求端口 IN_SERVICE:40900,管理员可见为什么绑不上。
		Respond(c, apitypes.CodeConflict, gin.H{"reason": err.Error()})
	case errors.Is(err, odn.ErrBindingNotFound):
		Respond(c, apitypes.CodeNotFound, nil)
	case errors.Is(err, odn.ErrNotServable):
		// 下单覆盖门控拒单(T12):40900 + 透传地址/状态,运营可见为什么拒。
		Respond(c, apitypes.CodeConflict, gin.H{"reason": err.Error()})
	case errors.Is(err, odn.ErrNoContractor),
		errors.Is(err, odn.ErrSettlementState):
		// 结算发起前置缺失/状态冲突(000204):40900 + 原因,管理员可见为什么拒。
		Respond(c, apitypes.CodeConflict, gin.H{"reason": err.Error()})
	case errors.Is(err, odn.ErrInvalidCode),
		errors.Is(err, odn.ErrGridMissing),
		errors.Is(err, odn.ErrGridFull),
		errors.Is(err, odn.ErrInvalidEndpoint),
		errors.Is(err, odn.ErrSamePriority),
		errors.Is(err, odn.ErrInvalidDeviceCode),
		errors.Is(err, odn.ErrBadHierarchy),
		errors.Is(err, odn.ErrInvalidInput):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, odn.ErrNotFound):
		Respond(c, apitypes.CodeNotFound, nil)
	case errors.Is(err, resource.ErrIllegalTransition),
		errors.Is(err, resource.ErrPortNotAvailable),
		errors.Is(err, aaa.ErrIllegalTransition),
		errors.Is(err, order.ErrIllegalTransition),
		errors.Is(err, provision.ErrIllegalTransition),
		errors.Is(err, asset.ErrIllegalTransition),
		errors.Is(err, billing.ErrIllegalReconTransition),
		errors.Is(err, billing.ErrIllegalInvoiceTransition),
		errors.Is(err, billing.ErrInvoiceNotTaxable),
		errors.Is(err, customer.ErrRegistrationConflict),
		errors.Is(err, partner.ErrApplicationConflict),
		errors.Is(err, partner.ErrNotPartner),
		errors.Is(err, partner.ErrStaffScope),
		errors.Is(err, partner.ErrApplicationDuplicate),
		errors.Is(err, customer.ErrRealNameConflict),
		errors.Is(err, customer.ErrRealNameMismatch),
		errors.Is(err, worker.ErrRegistrationConflict),
		errors.Is(err, worker.ErrRealNameConflict),
		errors.Is(err, worker.ErrInvalidReviewFields):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, worker.ErrScanBindRequired):
		// 状态机前置缺失(未扫码绑定不可激活,2026-09-04 任务A):40910,不再裸 50000。
		Respond(c, apitypes.CodeStateInvalid, nil)
	case errors.Is(err, worker.ErrGroupInvalid):
		// 班组快照不可用(工具借还等事实落库前置):40400 + 原因,操作员可自查归属。
		Respond(c, apitypes.CodeNotFound, gin.H{"reason": err.Error()})
	case errors.Is(err, loy.ErrConflict):
		// 积分域冲突(任务周期内已完成等):40900,不再裸 50000。
		Respond(c, apitypes.CodeConflict, nil)
	case errors.Is(err, quadlink.ErrAddressConflict):
		// 同地址活跃链路归属他客(重装复用守卫):40920 族 + 原因透传。
		Respond(c, apitypes.CodeScanMismatch, gin.H{"reason": err.Error()})
	case errors.Is(err, ai.ErrNotConfigured),
		errors.Is(err, ai.ErrInvalidInput):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, ai.ErrDownstream):
		Respond(c, apitypes.CodeDownstreamErr, nil)
	case errors.Is(err, portal.ErrSmsCooldown),
		errors.Is(err, order.ErrPartnerDailyCap),
		errors.Is(err, order.ErrPartnerCustomerCooldown),
		errors.Is(err, order.ErrDirectPhoneCap),
		errors.Is(err, order.ErrDirectAddressCap):
		Respond(c, apitypes.CodeResourceBusy, nil)
	case errors.Is(err, sms.ErrUnsupportedRegion):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, userdata.ErrContractDrift):
		// 列表契约漂移(主键列缺失/空):映射 500 并保留详细 message,
		// 调用方看到错误提示而不是 undefined 行(postmortem 0002 纵深防御)。
		Respond(c, apitypes.CodeInternal, gin.H{"reason": err.Error()})
	default:
		log.Printf("httpx: unmapped error (code=%d): %v", apitypes.CodeInternal, err)
		Respond(c, apitypes.CodeInternal, nil)
	}
}

// RespondOrderRiskBlocked 直营风控拦截统一出口:落 order.risk.blocked 审计 +
// 回 42300,reason 携带维度/计数/上限的可行动文案(只给"资源已被占用"时客服无从下手)。
// 返回 false 表示非风控拦截,调用方继续走 RespondErr,避免双写响应。
func RespondOrderRiskBlocked(a *app.Application, c *gin.Context, err error, customerID int64) bool {
	var ce *order.CapExceeded
	if !errors.As(err, &ce) {
		return false
	}
	RecordAudit(a, c, "order.risk.blocked", "customer",
		strconv.FormatInt(customerID, 10), map[string]any{"reason": err.Error()})
	Respond(c, apitypes.CodeResourceBusy, gin.H{"reason": riskBusyReason(ce)})
	return true
}

// riskBusyReason 面向操作员的中文指引;参数键可在 基础配置-业务参数 页直查调整。
func riskBusyReason(ce *order.CapExceeded) string {
	if ce.Kind == order.CapKindPhone {
		return fmt.Sprintf("该手机号24小时内已下单 %d 单(上限 %d),可明日再试或调大参数 risk.direct.phoneCap", ce.Count, ce.Cap)
	}
	return fmt.Sprintf("该地址在途订单已有 %d 笔(上限 %d),请先取消在途订单或调大参数 risk.direct.addressCap", ce.Count, ce.Cap)
}
