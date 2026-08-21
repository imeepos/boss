package httpx

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
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
	case errors.Is(err, user.ErrUsernameTaken),
		errors.Is(err, user.ErrDuplicate),
		errors.Is(err, user.ErrConflict),
		errors.Is(err, geo.ErrDuplicate),
		errors.Is(err, billing.ErrDuplicateInvoice):
		Respond(c, apitypes.CodeConflict, nil)
	case errors.Is(err, user.ErrInvalidInput),
		errors.Is(err, user.ErrRoleNotFound),
		errors.Is(err, user.ErrFKViolation),
		errors.Is(err, quadlink.ErrForeignKeyViolation),
		errors.Is(err, worker.ErrForeignKeyViolation),
		errors.Is(err, ErrGeoInvalidParam):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, user.ErrNotFound),
		errors.Is(err, resource.ErrNotFound),
		errors.Is(err, asset.ErrNotFound),
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
		errors.Is(err, portal.ErrNotFound):
		Respond(c, apitypes.CodeNotFound, nil)
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
		errors.Is(err, billing.ErrIllegalReconTransition),
		errors.Is(err, billing.ErrIllegalInvoiceTransition),
		errors.Is(err, billing.ErrInvoiceNotTaxable),
		errors.Is(err, customer.ErrRegistrationConflict),
		errors.Is(err, customer.ErrRealNameConflict),
		errors.Is(err, worker.ErrRegistrationConflict),
		errors.Is(err, worker.ErrRealNameConflict),
		errors.Is(err, worker.ErrInvalidReviewFields):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, ai.ErrNotConfigured),
		errors.Is(err, ai.ErrInvalidInput):
		Respond(c, apitypes.CodeInvalidParam, nil)
	case errors.Is(err, ai.ErrDownstream):
		Respond(c, apitypes.CodeDownstreamErr, nil)
	case errors.Is(err, portal.ErrSmsCooldown):
		Respond(c, apitypes.CodeResourceBusy, nil)
	case errors.Is(err, sms.ErrUnsupportedRegion):
		Respond(c, apitypes.CodeInvalidParam, nil)
	default:
		log.Printf("httpx: unmapped error (code=%d): %v", apitypes.CodeInternal, err)
		Respond(c, apitypes.CodeInternal, nil)
	}
}
