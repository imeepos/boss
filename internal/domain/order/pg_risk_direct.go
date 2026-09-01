package order

// 直营下单风控 v1(放量 FMS 前哨,2026-08-28):同手机号 24h 多单、同地址在途堆积拦截。
// 阈值经 biz_params 可调(risk.direct.*),读不到/非法值回退默认;关停用 risk.direct.enabled=false。
// 渠道链路已有独立风控(pg_partner_fraud.go),本文件只挂直营 submitRegular 路径。

import (
	"context"
	"fmt"
	"log"
	"strconv"
)

const (
	riskParamEnabled  = "risk.direct.enabled"
	riskParamPhoneCap = "risk.direct.phoneCap"
	riskParamAddrCap  = "risk.direct.addressCap"

	directPhoneCapDefault = 5
	directAddrCapDefault  = 3

	inFlightStatuses = "('PENDING','RESERVED','INSTALLING')"
)

// 拦截维度,httpapi 据此生成可行动文案。
const (
	CapKindPhone   = "phone"
	CapKindAddress = "address"
)

// CapExceeded 直营风控拦截:携带维度/当前计数/上限,handler 转成 42300+reason。
// 只回哨兵错误时客服只看到"资源已被占用",无从下手(2026-09-01 代客下单事故)。
type CapExceeded struct {
	Kind     string
	Count    int64
	Cap      int64
	sentinel error
}

func (e *CapExceeded) Error() string {
	return fmt.Sprintf("%s (count=%d cap=%d)", e.sentinel.Error(), e.Count, e.Cap)
}

func (e *CapExceeded) Unwrap() error { return e.sentinel }

// 构造器:sentinel 不导出,外部包(handler 测试/域外调用方)经此构造。
func NewPhoneCapExceeded(countVal, capVal int64) *CapExceeded {
	return &CapExceeded{Kind: CapKindPhone, Count: countVal, Cap: capVal, sentinel: ErrDirectPhoneCap}
}

func NewAddressCapExceeded(countVal, capVal int64) *CapExceeded {
	return &CapExceeded{Kind: CapKindAddress, Count: countVal, Cap: capVal, sentinel: ErrDirectAddressCap}
}

// riskParams 参数读取口径与佣金默认率一致:读失败/非法值回退默认,不阻塞下单。
func (s *PGStore) riskIntParam(ctx context.Context, key string, def int64) int64 {
	if s.params == nil {
		return def
	}
	v, err := s.params.GetParam(ctx, key)
	if err != nil {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func (s *PGStore) riskEnabled(ctx context.Context) bool {
	if s.params == nil {
		return true
	}
	v, err := s.params.GetParam(ctx, riskParamEnabled)
	if err != nil {
		return true
	}
	return v != "false"
}

// checkDirectRisk 提交前风控:任一超限返回对应哨兵错误(handler 落审计)。
func (s *PGStore) checkDirectRisk(ctx context.Context, req SubmitReq) error {
	if !s.riskEnabled(ctx) {
		return nil
	}
	phoneCap := s.riskIntParam(ctx, riskParamPhoneCap, directPhoneCapDefault)
	if phoneCap > 0 {
		var samePhone int64
		if err := s.db.QueryRow(ctx, `
			SELECT count(*) FROM orders o JOIN customers c ON c.id = o.customer_id
			WHERE c.phone = (SELECT phone FROM customers WHERE id = $1)
			  AND o.created_at >= now() - interval '24 hours'`,
			req.CustomerID).Scan(&samePhone); err != nil {
			return fmt.Errorf("order: risk phone count: %w", err)
		}
		if samePhone >= phoneCap {
			log.Printf("[order-risk] BLOCKED reason=PHONE_CAP customer=%d count=%d cap=%d", req.CustomerID, samePhone, phoneCap)
			return NewPhoneCapExceeded(samePhone, phoneCap)
		}
	}
	addrCap := s.riskIntParam(ctx, riskParamAddrCap, directAddrCapDefault)
	if addrCap > 0 {
		var inFlight int64
		if err := s.db.QueryRow(ctx, `
			SELECT count(*) FROM orders
			WHERE address_id = $1 AND status IN `+inFlightStatuses,
			req.AddressID).Scan(&inFlight); err != nil {
			return fmt.Errorf("order: risk address count: %w", err)
		}
		if inFlight >= addrCap {
			log.Printf("[order-risk] BLOCKED reason=ADDRESS_CAP address=%d count=%d cap=%d", req.AddressID, inFlight, addrCap)
			return NewAddressCapExceeded(inFlight, addrCap)
		}
	}
	return nil
}
