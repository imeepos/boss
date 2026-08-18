package userdata

import (
	"context"
	"errors"
	"fmt"
)

// usersSQL 用户列表:客户主档 join 账户/套餐/余额/欠费/在途业务(keyword 过滤)。
const usersSQL = `
SELECT c.id AS "customerId", c.name, c.phone, c.real_name_status AS "realNameStatus",
       c.service_status AS "serviceStatus",
       COALESCE(ua.login_name, '') AS "loginName",
       COALESCE(ua.auto_pay, FALSE) AS "autoPay",
       COALESCE(up.plan_name, '') AS "planName",
       COALESCE(ub.balance, 0) AS balance,
       COALESCE(ar.amount::float8, 0) AS "arrearsAmount",
       (SELECT count(*) FROM orders o WHERE o.customer_id = c.id AND o.status NOT IN ('DONE','CANCELLED')) AS "activeOrders"
FROM customers c
LEFT JOIN user_accounts ua ON ua.customer_id = c.id
LEFT JOIN LATERAL (SELECT plan_name FROM user_plans p WHERE p.customer_id = c.id ORDER BY p.id DESC LIMIT 1) up ON TRUE
LEFT JOIN user_balances ub ON ub.customer_id = c.id
LEFT JOIN LATERAL (SELECT amount FROM arrears a WHERE a.customer_id = c.id LIMIT 1) ar ON TRUE
WHERE ($1 = '' OR c.name ILIKE '%' || $1 || '%' OR c.phone ILIKE '%' || $1 || '%')
ORDER BY c.id`

// ListUsers 用户列表(keyword 过滤姓名/手机号)。
func (s *PGStore) ListUsers(ctx context.Context, keyword string) ([]map[string]any, error) {
	return s.listMaps(ctx, usersSQL, keyword)
}

// GetUserDetail 用户详情聚合:该用户在用户端可见的全部数据。
func (s *PGStore) GetUserDetail(ctx context.Context, customerID int64) (map[string]any, error) {
	cust, err := s.getMap(ctx, `SELECT id AS "customerId", name, phone, id_type AS "idType",
		id_no AS "idNo", real_name_status AS "realNameStatus", service_status AS "serviceStatus",
		region_name AS "regionName", created_at AS "createdAt"
		FROM customers WHERE id = $1`, customerID)
	if err != nil {
		return nil, fmt.Errorf("userdata: get user detail: %w", err)
	}
	pick := func(list []map[string]any, err error) []map[string]any {
		if err != nil || list == nil {
			return []map[string]any{}
		}
		return list
	}
	cust["notify"], err = s.getMap(ctx,
		`SELECT business, marketing, channel FROM user_notify_settings WHERE customer_id = $1`, customerID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	cust["addresses"] = pick(s.listMaps(ctx,
		`SELECT id, addr_code AS "addrCode", contact, phone, detail, is_default AS "isDefault"
		 FROM user_addresses WHERE customer_id = $1 ORDER BY id`, customerID))
	cust["balances"] = pick(s.listMaps(ctx,
		`SELECT balance, warn_line AS "warnLine", (balance < warn_line) AS "lowWarn"
		 FROM user_balances WHERE customer_id = $1`, customerID))
	cust["plans"] = pick(s.listMaps(ctx,
		`SELECT id, product_id AS "productId", plan_name AS "planName", status, effective_at AS "effectiveAt"
		 FROM user_plans WHERE customer_id = $1 ORDER BY id`, customerID))
	cust["addons"] = pick(s.listMaps(ctx,
		`SELECT s.addon_id AS "addonId", a.name, s.action, s.created_at AS "createdAt"
		 FROM addon_subscriptions s JOIN addons a ON a.addon_id = s.addon_id
		 WHERE s.customer_id = $1 ORDER BY s.id`, customerID))
	cust["usages"] = pick(s.listMaps(ctx,
		`SELECT month, upload_gb AS "uploadGb", download_gb AS "downloadGb", total_gb AS "totalGb"
		 FROM user_usages WHERE customer_id = $1 ORDER BY month DESC`, customerID))
	cust["messages"] = pick(s.listMaps(ctx,
		`SELECT id, type, title, content, read, created_at AS "createdAt"
		 FROM user_messages WHERE customer_id = $1 ORDER BY id DESC`, customerID))
	cust["coupons"] = pick(s.listMaps(ctx,
		`SELECT coupon_id AS "couponId", name, amount, status, expire_at AS "expireAt"
		 FROM coupons WHERE customer_id = $1 ORDER BY coupon_id`, customerID))
	cust["complaints"] = pick(s.listMaps(ctx,
		`SELECT complaint_id AS "complaintId", type, content, status
		 FROM user_complaints WHERE customer_id = $1 ORDER BY complaint_id`, customerID))
	cust["verifyRecords"] = pick(s.listMaps(ctx,
		`SELECT id, step, result, created_at AS "createdAt"
		 FROM user_verify_records WHERE customer_id = $1 ORDER BY id`, customerID))
	cust["bills"] = pick(s.listMaps(ctx,
		`SELECT id, bill_no AS "billNo", amount::float8 AS amount, status, period
		 FROM bills WHERE customer_id = $1 ORDER BY id DESC`, customerID))
	cust["payments"] = pick(s.listMaps(ctx,
		`SELECT p.id, b.bill_no AS "billNo", p.method AS channel, p.amount::float8 AS amount, p.created_at AS "paidAt"
		 FROM payments p JOIN bills b ON b.id = p.bill_id WHERE b.customer_id = $1 ORDER BY p.id DESC`, customerID))
	cust["invoices"] = pick(s.listMaps(ctx,
		`SELECT id, bill_no AS "billNo", invoice_no AS "invoiceNo", amount, title, status
		 FROM user_invoices WHERE customer_id = $1 ORDER BY id DESC`, customerID))
	cust["orders"] = pick(s.listMaps(ctx,
		`SELECT id, order_no AS "orderNo", stage, status, created_at AS "createdAt"
		 FROM orders WHERE customer_id = $1 ORDER BY id DESC`, customerID))
	cust["faults"] = pick(s.listMaps(ctx,
		`SELECT id, ticket_no AS "ticketNo", type, status
		 FROM complaints WHERE customer_id = $1 ORDER BY id DESC`, customerID))
	return cust, nil
}
