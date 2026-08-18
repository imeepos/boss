package userdata

import "context"

// 用户端实体表/全局配置的只读管理视图(列名即 JSON 字段)。

func (s *PGStore) ListUserAccounts(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT ua.id, ua.customer_id AS "customerId", c.name AS "customerName",
		ua.login_name AS "loginName", ua.registered_at AS "registeredAt",
		ua.password_updated_at AS "passwordUpdatedAt", ua.auto_pay AS "autoPay"
		FROM user_accounts ua JOIN customers c ON c.id = ua.customer_id ORDER BY ua.id`)
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

func (s *PGStore) ListNotifySettings(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT n.id, n.customer_id AS "customerId", c.name AS "customerName",
		n.business, n.marketing, n.channel
		FROM user_notify_settings n JOIN customers c ON c.id = n.customer_id ORDER BY n.id`)
}

func (s *PGStore) ListUserFaqs(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT faq_id AS "faqId", category, question, answer, active
		FROM user_faqs ORDER BY faq_id`)
}

func (s *PGStore) ListUserMessages(ctx context.Context, keyword string) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT id, customer_id AS "customerId", type, title, content, read,
		created_at AS "createdAt" FROM user_messages
		WHERE ($1 = '' OR title ILIKE '%' || $1 || '%' OR content ILIKE '%' || $1 || '%')
		ORDER BY id DESC`, keyword)
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

func (s *PGStore) ListUserBalances(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT b.customer_id AS "customerId", c.name AS "customerName",
		b.balance, b.warn_line AS "warnLine", (b.balance < b.warn_line) AS "lowWarn"
		FROM user_balances b JOIN customers c ON c.id = b.customer_id ORDER BY b.customer_id`)
}

func (s *PGStore) ListTopupDenominations(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT denom_id AS "denomId", amount, bonus, active
		FROM topup_denominations ORDER BY amount`)
}

func (s *PGStore) ListUserInvoices(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT id, customer_id AS "customerId", bill_no AS "billNo",
		invoice_no AS "invoiceNo", amount, title, status FROM user_invoices ORDER BY id DESC`)
}

func (s *PGStore) ListUserComplaints(ctx context.Context) ([]map[string]any, error) {
	return s.listMaps(ctx, `SELECT complaint_id AS "complaintId", customer_id AS "customerId",
		type, content, status FROM user_complaints ORDER BY complaint_id`)
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
