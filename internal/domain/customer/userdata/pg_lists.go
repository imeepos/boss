package userdata

import "context"

// 用户端实体表/全局配置的只读管理视图(列名即 JSON 字段)。
// 双胞胎表(余额/消息/通知偏好/发票/投诉)已按裁定 D1(2026-08-20)改为读权威表的薄适配层:
// portal_wallets/portal_messages/portal_prefs/invoices/complaints,user_* 表不再被读。

// Deprecated: autoPay 权威态在 portal_billing_prefs;user_accounts.auto_pay 列已停用。
func (s *PGStore) ListUserAccounts(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT ua.id, ua.customer_id AS "customerId", c.name AS "customerName",
		ua.login_name AS "loginName", ua.registered_at AS "registeredAt",
		ua.password_updated_at AS "passwordUpdatedAt",
		COALESCE(bp.auto_pay_enabled, FALSE) AS "autoPay"
		FROM user_accounts ua
		JOIN customers c ON c.id = ua.customer_id
		LEFT JOIN portal_billing_prefs bp ON bp.customer_id = ua.customer_id
		ORDER BY ua.id`)
}

func (s *PGStore) ListUserAddresses(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT id, customer_id AS "customerId", addr_code AS "addrCode",
		contact, phone, detail, is_default AS "isDefault" FROM user_addresses ORDER BY id`)
}

func (s *PGStore) ListUserPlans(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT p.id, p.customer_id AS "customerId", p.product_id AS "productId",
		p.plan_name AS "planName", p.status, p.effective_at AS "effectiveAt"
		FROM user_plans p ORDER BY p.id`)
}

func (s *PGStore) ListAddons(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT addon_id AS "addonId", name, price, status,
		subscriber_count AS "subscriberCount" FROM addons ORDER BY addon_id`)
}

func (s *PGStore) ListAddonSubscriptions(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT s.id, s.customer_id AS "customerId", s.addon_id AS "addonId",
		a.name AS "addonName", s.action, s.created_at AS "createdAt"
		FROM addon_subscriptions s JOIN addons a ON a.addon_id = s.addon_id ORDER BY s.id`)
}

// Deprecated: 通知偏好权威态在 portal_prefs.notify;user_notify_settings 已停用。
func (s *PGStore) ListNotifySettings(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT p.customer_id AS "customerId", c.name AS "customerName",
		COALESCE((p.notify->>'business')::boolean, FALSE) AS business,
		COALESCE((p.notify->>'marketing')::boolean, FALSE) AS marketing,
		COALESCE(p.notify->>'channel', 'app') AS channel
		FROM portal_prefs p JOIN customers c ON c.id = p.customer_id ORDER BY p.customer_id`)
}

func (s *PGStore) ListUserFaqs(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT faq_id AS "faqId", category, question, answer, active
		FROM user_faqs ORDER BY faq_id`)
}

// Deprecated: 消息权威态在 portal_messages(payload JSONB);user_messages 已停用。
func (s *PGStore) ListUserMessages(ctx context.Context, keyword string) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT m.id, m.customer_id AS "customerId",
		COALESCE(m.payload->>'type', 'notice') AS type,
		m.payload->>'title' AS title, m.payload->>'content' AS content, m.read,
		m.created_at AS "createdAt" FROM portal_messages m
		WHERE ($1 = '' OR m.payload->>'title' ILIKE '%' || $1 || '%'
			OR m.payload->>'content' ILIKE '%' || $1 || '%')
		ORDER BY m.id DESC`, keyword)
}

func (s *PGStore) ListCoupons(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT coupon_id AS "couponId", customer_id AS "customerId", name,
		amount, status, expire_at AS "expireAt" FROM coupons ORDER BY coupon_id`)
}

func (s *PGStore) GetInviteConfig(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT id, invite_link AS "inviteLink", reward_amount AS "rewardAmount",
		active FROM invite_config ORDER BY id`)
}

func (s *PGStore) ListUserUsages(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT id, customer_id AS "customerId", month,
		upload_gb AS "uploadGb", download_gb AS "downloadGb", total_gb AS "totalGb"
		FROM user_usages ORDER BY customer_id, month DESC`)
}

func (s *PGStore) ListDiyGuides(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT guide_id AS "guideId", title, category, steps, active
		FROM diy_guides ORDER BY guide_id`)
}

func (s *PGStore) ListAgreements(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT agreement_id AS "agreementId", type, version, content,
		effective_at AS "effectiveAt" FROM agreements ORDER BY agreement_id`)
}

// Deprecated: 余额权威态在 portal_wallets;user_balances 已停用(portal_wallets 无 warn_line,warnLine 恒 0)。
func (s *PGStore) ListUserBalances(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT w.customer_id AS "customerId", c.name AS "customerName",
		w.balance::float8 AS balance, 0::bigint AS "warnLine", (w.balance < 0) AS "lowWarn"
		FROM portal_wallets w JOIN customers c ON c.id = w.customer_id ORDER BY w.customer_id`)
}

func (s *PGStore) ListTopupDenominations(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT denom_id AS "denomId", amount, bonus, active
		FROM topup_denominations ORDER BY amount`)
}

// Deprecated: 发票权威态在 invoices(billing 域);user_invoices 已停用。
func (s *PGStore) ListUserInvoices(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT i.id, i.customer_id AS "customerId", i.bill_no AS "billNo",
		i.invoice_no AS "invoiceNo", i.total_amount::float8 AS amount, i.title, i.status
		FROM invoices i ORDER BY i.id DESC`)
}

// Deprecated: 投诉权威态在 complaints(order 域);user_complaints 已停用(content 以 ticket_no 代)。
func (s *PGStore) ListUserComplaints(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT cp.id::text AS "complaintId", cp.customer_id AS "customerId",
		cp.type, cp.ticket_no AS content, cp.status FROM complaints cp ORDER BY cp.id`)
}

func (s *PGStore) ListUserVerifyRecords(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT id, customer_id AS "customerId", step, result,
		created_at AS "createdAt" FROM user_verify_records ORDER BY id`)
}

func (s *PGStore) ListProductSpecs(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT product_id AS "productId", highlights, specs
		FROM product_specs ORDER BY product_id`)
}

func (s *PGStore) ListUserBillItems(ctx context.Context, billNo string) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT id, bill_no AS "billNo", customer_id AS "customerId",
		name, amount FROM user_bill_items WHERE ($1 = '' OR bill_no = $1) ORDER BY id`, billNo)
}
