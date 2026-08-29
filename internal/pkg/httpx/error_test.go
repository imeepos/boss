package httpx

import (
	"errors"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/sms"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// TestRespondErrMapping 逐条覆盖 RespondErr 的全部哨兵错误映射。
func TestRespondErrMapping(t *testing.T) {
	cases := []struct {
		err  error
		code apitypes.Code
	}{
		{user.ErrUnauthorized, apitypes.CodeUnauthorized},
		{user.ErrUsernameTaken, apitypes.CodeConflict},
		{user.ErrDuplicate, apitypes.CodeConflict},
		{user.ErrConflict, apitypes.CodeConflict},
		{customer.ErrDuplicate, apitypes.CodeConflict},
		{geo.ErrDuplicate, apitypes.CodeConflict},
		{billing.ErrDuplicateInvoice, apitypes.CodeConflict},
		{user.ErrInvalidInput, apitypes.CodeInvalidParam},
		{user.ErrRoleNotFound, apitypes.CodeInvalidParam},
		{user.ErrFKViolation, apitypes.CodeInvalidParam},
		{billing.ErrForeignKeyViolation, apitypes.CodeInvalidParam},
		{billing.ErrInvalidMethod, apitypes.CodeInvalidParam},
		{ErrGeoInvalidParam, apitypes.CodeInvalidParam},
		{provision.ErrForeignKeyViolation, apitypes.CodeInvalidParam},
		{user.ErrNotFound, apitypes.CodeNotFound},
		{resource.ErrNotFound, apitypes.CodeNotFound},
		{asset.ErrNotFound, apitypes.CodeNotFound},
		{customer.ErrCustomerNotFound, apitypes.CodeNotFound},
		{customer.ErrProductNotFound, apitypes.CodeNotFound},
		{order.ErrOrderNotFound, apitypes.CodeNotFound},
		{billing.ErrNotFound, apitypes.CodeNotFound},
		{geo.ErrNotFound, apitypes.CodeNotFound},
		{provision.ErrTaskNotFound, apitypes.CodeNotFound},
		{customer.ErrRegistrationNotFound, apitypes.CodeNotFound},
		{customer.ErrRealNameNotFound, apitypes.CodeNotFound},
		{worker.ErrNotFound, apitypes.CodeNotFound},
		{worker.ErrRegistrationNotFound, apitypes.CodeNotFound},
		{worker.ErrRealNameNotFound, apitypes.CodeNotFound},
		{userdata.ErrNotFound, apitypes.CodeNotFound},
		{portal.ErrNotFound, apitypes.CodeNotFound},
		{resource.ErrIllegalTransition, apitypes.CodeInvalidParam},
		{resource.ErrPortNotAvailable, apitypes.CodeInvalidParam},
		{order.ErrIllegalTransition, apitypes.CodeInvalidParam},
		{provision.ErrIllegalTransition, apitypes.CodeInvalidParam},
		{billing.ErrIllegalReconTransition, apitypes.CodeInvalidParam},
		{billing.ErrIllegalInvoiceTransition, apitypes.CodeInvalidParam},
		{billing.ErrInvoiceNotTaxable, apitypes.CodeInvalidParam},
		{customer.ErrRegistrationConflict, apitypes.CodeInvalidParam},
		{customer.ErrRealNameConflict, apitypes.CodeInvalidParam},
		{worker.ErrRegistrationConflict, apitypes.CodeInvalidParam},
		{worker.ErrRealNameConflict, apitypes.CodeInvalidParam},
		{ai.ErrNotConfigured, apitypes.CodeInvalidParam},
		{ai.ErrInvalidInput, apitypes.CodeInvalidParam},
		{ai.ErrDownstream, apitypes.CodeDownstreamErr},
		{portal.ErrSmsCooldown, apitypes.CodeResourceBusy},
		{sms.ErrUnsupportedRegion, apitypes.CodeInvalidParam},
		{asset.ErrBindingConflict, apitypes.CodeConflict},
		{errors.New("boom"), apitypes.CodeInternal},
	}
	for _, tc := range cases {
		if got := respondCode(func(c *gin.Context) { RespondErr(c, tc.err) }); got != tc.code {
			t.Errorf("err %v: code = %d, want %d", tc.err, got, tc.code)
		}
	}
}
