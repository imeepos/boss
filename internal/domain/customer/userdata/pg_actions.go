package userdata

import (
	"context"
	"fmt"
	"time"
)

// strOrNil 空串归一为 NULL(可空时间戳入参约定)。
func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// defaultStr 空串取默认值。
func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (s *PGStore) UpdateUserAccount(ctx context.Context, customerID int64, u UserAccountUpdate) error {
	return s.execAffected(ctx, "update user account",
		`UPDATE user_accounts SET auto_pay = $2 WHERE customer_id = $1`, customerID, u.AutoPay)
}

func (s *PGStore) CreateUserAddress(ctx context.Context, a UserAddress) (int64, error) {
	return s.insertReturning(ctx, "create user address", `
		INSERT INTO user_addresses(customer_id, addr_code, contact, phone, detail, is_default)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		a.CustomerID, a.AddrCode, a.Contact, a.Phone, a.Detail, a.IsDefault)
}

func (s *PGStore) CreateUserPlan(ctx context.Context, p UserPlan) (int64, error) {
	return s.insertReturning(ctx, "create user plan", `
		INSERT INTO user_plans(customer_id, product_id, plan_name, status, effective_at)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		p.CustomerID, p.ProductID, p.PlanName, defaultStr(p.Status, "ACTIVE"), strOrNil(p.EffectiveAt))
}

func (s *PGStore) CreateAddon(ctx context.Context, a Addon) error {
	id := defaultStr(a.AddonID, fmt.Sprintf("ADD-%x", time.Now().UnixNano()))
	return s.execAffected(ctx, "create addon", `
		INSERT INTO addons(addon_id, name, price, status) VALUES($1,$2,$3,$4)`,
		id, a.Name, a.Price, defaultStr(a.Status, "on"))
}

func (s *PGStore) ToggleAddon(ctx context.Context, addonID string) error {
	return s.execAffected(ctx, "toggle addon",
		`UPDATE addons SET status = CASE WHEN status = 'on' THEN 'off' ELSE 'on' END WHERE addon_id = $1`, addonID)
}

func (s *PGStore) CreateAddonSubscription(ctx context.Context, sub AddonSubscription) (int64, error) {
	id, err := s.insertReturning(ctx, "create addon subscription", `
		INSERT INTO addon_subscriptions(customer_id, addon_id, action) VALUES($1,$2,$3) RETURNING id`,
		sub.CustomerID, sub.AddonID, sub.Action)
	if err != nil {
		return 0, err
	}
	if sub.Action == "subscribe" {
		return id, s.execAffected(ctx, "bump addon count",
			`UPDATE addons SET subscriber_count = subscriber_count + 1 WHERE addon_id = $1`, sub.AddonID)
	}
	return id, nil
}

func (s *PGStore) UpdateNotifySettings(ctx context.Context, customerID int64, n NotifySetting) error {
	return s.execAffected(ctx, "update notify settings", `
		INSERT INTO user_notify_settings(customer_id, business, marketing, channel)
		VALUES($1,$2,$3,$4)
		ON CONFLICT (customer_id) DO UPDATE SET business = $2, marketing = $3, channel = $4`,
		customerID, n.Business, n.Marketing, defaultStr(n.Channel, "app"))
}

func (s *PGStore) CreateUserFaq(ctx context.Context, f UserFaq) error {
	return s.execAffected(ctx, "create user faq", `
		INSERT INTO user_faqs(faq_id, category, question, answer, active) VALUES($1,$2,$3,$4,$5)`,
		defaultStr(f.FaqID, fmt.Sprintf("FAQ-%s", f.Category)), f.Category, f.Question, f.Answer, f.Active)
}

func (s *PGStore) ToggleUserFaq(ctx context.Context, faqID string) error {
	return s.execAffected(ctx, "toggle user faq",
		`UPDATE user_faqs SET active = NOT active WHERE faq_id = $1`, faqID)
}

func (s *PGStore) CreateUserMessage(ctx context.Context, m UserMessage) (int64, error) {
	return s.insertReturning(ctx, "create user message", `
		INSERT INTO user_messages(customer_id, type, title, content) VALUES($1,$2,$3,$4) RETURNING id`,
		m.CustomerID, defaultStr(m.Type, "notice"), m.Title, m.Content)
}

func (s *PGStore) MarkAllMessagesRead(ctx context.Context, customerID int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE user_messages SET read = TRUE WHERE customer_id = $1`, customerID)
	if err != nil {
		return fmt.Errorf("userdata: mark all read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PGStore) CreateCoupon(ctx context.Context, cp Coupon) error {
	return s.execAffected(ctx, "create coupon", `
		INSERT INTO coupons(coupon_id, customer_id, name, amount, expire_at) VALUES($1,$2,$3,$4,$5)`,
		defaultStr(cp.CouponID, fmt.Sprintf("CPN-%d", cp.CustomerID)), cp.CustomerID, cp.Name, cp.Amount, strOrNil(cp.ExpireAt))
}

func (s *PGStore) DisableCoupon(ctx context.Context, couponID string) error {
	return s.execAffected(ctx, "disable coupon",
		`UPDATE coupons SET status = 'disabled' WHERE coupon_id = $1`, couponID)
}

func (s *PGStore) ToggleDiyGuide(ctx context.Context, guideID string) error {
	return s.execAffected(ctx, "toggle diy guide",
		`UPDATE diy_guides SET active = NOT active WHERE guide_id = $1`, guideID)
}

func (s *PGStore) UpdateAgreement(ctx context.Context, agreementID string, a Agreement) error {
	return s.execAffected(ctx, "update agreement", `
		UPDATE agreements SET type = $2, version = $3, content = $4,
			effective_at = COALESCE($5, effective_at) WHERE agreement_id = $1`,
		agreementID, a.Type, a.Version, a.Content, strOrNil(a.EffectiveAt))
}

func (s *PGStore) AdjustUserBalance(ctx context.Context, customerID int64, delta int64) error {
	return s.execAffected(ctx, "adjust user balance", `
		INSERT INTO user_balances(customer_id, balance, warn_line) VALUES($1, $2, 0)
		ON CONFLICT (customer_id) DO UPDATE SET balance = user_balances.balance + $2`,
		customerID, delta)
}

func (s *PGStore) UpdateTopupDenomination(ctx context.Context, denomID string, d TopupDenomination) error {
	return s.execAffected(ctx, "update topup denomination", `
		UPDATE topup_denominations SET amount = $2, bonus = $3, active = $4 WHERE denom_id = $1`,
		denomID, d.Amount, d.Bonus, d.Active)
}

func (s *PGStore) CreateUserInvoice(ctx context.Context, inv UserInvoice) (int64, error) {
	return s.insertReturning(ctx, "create user invoice", `
		INSERT INTO user_invoices(customer_id, bill_no, invoice_no, amount, title) VALUES($1,$2,$3,$4,$5) RETURNING id`,
		inv.CustomerID, inv.BillNo, inv.InvoiceNo, inv.Amount, inv.Title)
}

func (s *PGStore) CloseUserComplaint(ctx context.Context, complaintID string) error {
	return s.execAffected(ctx, "close user complaint",
		`UPDATE user_complaints SET status = 'CLOSED' WHERE complaint_id = $1`, complaintID)
}

func (s *PGStore) UpdateProductSpec(ctx context.Context, productID string, p ProductSpec) error {
	return s.execAffected(ctx, "update product spec", `
		INSERT INTO product_specs(product_id, highlights, specs) VALUES($1,$2,$3)
		ON CONFLICT (product_id) DO UPDATE SET highlights = $2, specs = $3`,
		productID, p.Highlights, p.Specs)
}
