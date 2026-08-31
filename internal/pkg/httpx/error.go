package httpx

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/backup"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/cms"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/domain/metric"
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
	switch {
	case errors.Is(err, user.ErrUnauthorized):
		Respond(c, apitypes.CodeUnauthorized, nil)
	case errors.Is(err, user.ErrRoleProtected),
		errors.Is(err, partner.ErrRegionOutsideEnterprise):
		Respond(c, apitypes.CodeForbidden, nil)
	case errors.Is(err, user.ErrUsernameTaken),
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
		errors.Is(err, worker.ErrDuplicate):
		// 资产/标签双绑冲突:40900 + 透传 err.Error()(含具体资产/标签 id),
		// 调用方能区分"标签已绑"vs"资产已绑",与 40920 扫码不一致明确区分。
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
		errors.Is(err, procurement.ErrForeignKey),
		errors.Is(err, procurement.ErrInvalidTransition),
		errors.Is(err, order.ErrInstallInput),
		errors.Is(err, backup.ErrInvalidInput),
		errors.Is(err, order.ErrInvalidInput),
		errors.Is(err, billing.ErrInvalidMethod),
		errors.Is(err, billing.ErrForeignKeyViolation),
		errors.Is(err, ErrGeoInvalidParam),
		errors.Is(err, cms.ErrInvalidPost),
		errors.Is(err, cms.ErrInvalidCategory):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, user.ErrNotFound),
		errors.Is(err, resource.ErrNotFound),
		errors.Is(err, asset.ErrNotFound),
		errors.Is(err, procurement.ErrNotFound),
		errors.Is(err, customer.ErrCustomerNotFound),
		errors.Is(err, customer.ErrProductNotFound),
		errors.Is(err, order.ErrOrderNotFound),
		errors.Is(err, billing.ErrNotFound),
		errors.Is(err, geo.ErrNotFound),
		errors.Is(err, provision.ErrTaskNotFound),
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
		errors.Is(err, cms.ErrCategoryNotFound):
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
	case errors.Is(err, odn.ErrInvalidCode),
		errors.Is(err, odn.ErrGridMissing),
		errors.Is(err, odn.ErrGridFull),
		errors.Is(err, odn.ErrInvalidEndpoint),
		errors.Is(err, odn.ErrSamePriority),
		errors.Is(err, odn.ErrInvalidDeviceCode),
		errors.Is(err, odn.ErrBadHierarchy):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, odn.ErrNotFound):
		Respond(c, apitypes.CodeNotFound, nil)
	case errors.Is(err, resource.ErrIllegalTransition),
		errors.Is(err, resource.ErrPortNotAvailable),
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
