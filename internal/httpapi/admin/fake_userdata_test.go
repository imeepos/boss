package adminapi

// fakeUserdata 桩 userdata.Service:所有方法成功返回可断言的最小数据。

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/customer/userdata"
)

type fakeUserdata struct {
	row        map[string]any
	calledPath string
}

func (f *fakeUserdata) ok() []map[string]any {
	return []map[string]any{f.row}
}

func (f *fakeUserdata) ListUsers(context.Context, string) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) GetUserDetail(context.Context, int64) (map[string]any, error) {
	return f.row, nil
}
func (f *fakeUserdata) UpdateUserAccount(context.Context, int64, userdata.UserAccountUpdate) error {
	return nil
}
func (f *fakeUserdata) ListUserAccounts(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) ListUserAddresses(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) CreateUserAddress(context.Context, userdata.UserAddress) (int64, error) {
	return 1, nil
}
func (f *fakeUserdata) ListUserPlans(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) CreateUserPlan(context.Context, userdata.UserPlan) (int64, error) {
	return 1, nil
}
func (f *fakeUserdata) ListAddons(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) CreateAddon(context.Context, userdata.Addon) error { return nil }
func (f *fakeUserdata) ToggleAddon(_ context.Context, addonID string) error {
	if addonID == "ADD-404" {
		return userdata.ErrNotFound
	}
	f.calledPath = addonID
	return nil
}
func (f *fakeUserdata) ListAddonSubscriptions(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) CreateAddonSubscription(context.Context, userdata.AddonSubscription) (int64, error) {
	return 1, nil
}
func (f *fakeUserdata) ListNotifySettings(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) UpdateNotifySettings(context.Context, int64, userdata.NotifySetting) error {
	return nil
}
func (f *fakeUserdata) ListUserFaqs(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) CreateUserFaq(context.Context, userdata.UserFaq) error { return nil }
func (f *fakeUserdata) ToggleUserFaq(_ context.Context, faqID string) error {
	f.calledPath = faqID
	return nil
}
func (f *fakeUserdata) ListUserMessages(context.Context, string) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) CreateUserMessage(context.Context, userdata.UserMessage) (int64, error) {
	return 1, nil
}
func (f *fakeUserdata) MarkAllMessagesRead(context.Context, int64) error { return nil }
func (f *fakeUserdata) ListCoupons(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) CreateCoupon(context.Context, userdata.Coupon) error { return nil }
func (f *fakeUserdata) DisableCoupon(_ context.Context, couponID string) error {
	f.calledPath = couponID
	return nil
}
func (f *fakeUserdata) GetInviteConfig(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) ListUserUsages(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) ListDiyGuides(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) ToggleDiyGuide(_ context.Context, guideID string) error {
	f.calledPath = guideID
	return nil
}
func (f *fakeUserdata) ListAgreements(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) UpdateAgreement(_ context.Context, agreementID string, _a userdata.Agreement) error {
	f.calledPath = agreementID
	return nil
}
func (f *fakeUserdata) ListUserBalances(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) AdjustUserBalance(context.Context, int64, int64) error { return nil }
func (f *fakeUserdata) ListTopupDenominations(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) UpdateTopupDenomination(_ context.Context, denomID string, _d userdata.TopupDenomination) error {
	f.calledPath = denomID
	return nil
}
func (f *fakeUserdata) ListUserInvoices(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) CreateUserInvoice(context.Context, userdata.UserInvoice) (int64, error) {
	return 1, nil
}
func (f *fakeUserdata) ListUserComplaints(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) CloseUserComplaint(_ context.Context, complaintID string) error {
	f.calledPath = complaintID
	return nil
}
func (f *fakeUserdata) ListUserVerifyRecords(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) ListProductSpecs(context.Context) ([]map[string]any, error) {
	return f.ok(), nil
}
func (f *fakeUserdata) UpdateProductSpec(_ context.Context, productID string, _p userdata.ProductSpec) error {
	f.calledPath = productID
	return nil
}
func (f *fakeUserdata) ListUserBillItems(context.Context, string) ([]map[string]any, error) {
	return f.ok(), nil
}
