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

// Deprecated: 自动缴费权威态在 portal_billing_prefs;不再写 user_accounts.auto_pay(裁定 D1)。
func (s *PGStore) UpdateUserAccount(ctx context.Context, customerID int64, u UserAccountUpdate) error {
	_, err := s.db.Exec(ctx, `INSERT INTO portal_billing_prefs(customer_id, auto_pay_enabled)
		VALUES($1, $2)
		ON CONFLICT (customer_id) DO UPDATE SET auto_pay_enabled = $2, updated_at = now()`,
		customerID, u.AutoPay)
	return err
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

// Deprecated: 通知偏好权威态在 portal_prefs.notify;不再写 user_notify_settings(裁定 D1)。
func (s *PGStore) UpdateNotifySettings(ctx context.Context, customerID int64, n NotifySetting) error {
	tag, err := s.db.Exec(ctx, `INSERT INTO portal_prefs(customer_id, notify)
		VALUES($1, jsonb_build_object('business', $2, 'marketing', $3, 'channel', $4))
		ON CONFLICT (customer_id) DO UPDATE SET notify = EXCLUDED.notify`,
		customerID, n.Business, n.Marketing, defaultStr(n.Channel, "app"))
	if err != nil {
		return fmt.Errorf("userdata: update notify settings: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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

// Deprecated: 消息权威态在 portal_messages;不再写 user_messages(裁定 D1)。
func (s *PGStore) CreateUserMessage(ctx context.Context, m UserMessage) (int64, error) {
	return s.insertReturning(ctx, "create user message", `
		INSERT INTO portal_messages(customer_id, payload)
		VALUES($1, jsonb_build_object('type', $2, 'title', $3, 'content', $4)) RETURNING id`,
		m.CustomerID, defaultStr(m.Type, "notice"), m.Title, m.Content)
}

// Deprecated: 消息权威态在 portal_messages;不再写 user_messages(裁定 D1)。
func (s *PGStore) MarkAllMessagesRead(ctx context.Context, customerID int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE portal_messages SET read = TRUE WHERE customer_id = $1`, customerID)
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

// Deprecated: 余额权威态在 portal_wallets;不再写 user_balances(裁定 D1)。
func (s *PGStore) AdjustUserBalance(ctx context.Context, customerID int64, delta int64) error {
	tag, err := s.db.Exec(ctx, `INSERT INTO portal_wallets(customer_id, balance) VALUES($1, $2)
		ON CONFLICT (customer_id) DO UPDATE SET balance = portal_wallets.balance + EXCLUDED.balance`,
		customerID, delta)
	if err != nil {
		return fmt.Errorf("userdata: adjust user balance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PGStore) UpdateTopupDenomination(ctx context.Context, denomID string, d TopupDenomination) error {
	return s.execAffected(ctx, "update topup denomination", `
		UPDATE topup_denominations SET amount = $2, bonus = $3, active = $4 WHERE denom_id = $1`,
		denomID, d.Amount, d.Bonus, d.Active)
}

// ErrWriteStopped 写路径已按裁定 D1 停写,须改走对应权威域。
var ErrWriteStopped = errWriteStopped{}

type errWriteStopped struct{}

func (errWriteStopped) Error() string {
	return "userdata: 写路径已停用(裁定 D1,见 docs/notes/adopted/2026-08-20-db-dualtrack-convergence.md)"
}

// Deprecated: 发票权威态在 invoices(billing 域);user_invoices 停写。
// 二阶段: 开票走 billing.TaxService(需 bill 归属校验与 ARN 发号,无法直插)。
func (s *PGStore) CreateUserInvoice(ctx context.Context, inv UserInvoice) (int64, error) {
	_ = inv
	return 0, ErrWriteStopped
}

// Deprecated: 投诉权威态在 complaints(order 域);不再写 user_complaints(裁定 D1)。
// 寻址兼容旧 complaint_id 字符串:按 complaints.id 文本或 ticket_no 命中。
func (s *PGStore) CloseUserComplaint(ctx context.Context, complaintID string) error {
	return s.execAffected(ctx, "close user complaint",
		`UPDATE complaints SET status = 'CLOSED' WHERE ticket_no = $1 OR id::text = $1`, complaintID)
}

func (s *PGStore) UpdateProductSpec(ctx context.Context, productID string, p ProductSpec) error {
	return s.execAffected(ctx, "update product spec", `
		INSERT INTO product_specs(product_id, highlights, specs) VALUES($1,$2,$3)
		ON CONFLICT (product_id) DO UPDATE SET highlights = $2, specs = $3`,
		productID, p.Highlights, p.Specs)
}
