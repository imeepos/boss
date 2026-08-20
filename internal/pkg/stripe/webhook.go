package stripe

// Stripe webhook:签名校验(v1 scheme)+ 事件解析。
// 记账事实源在本系统(payments),webhook 只做"收单回调",幂等由 pay_no 唯一约束兜底。

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Webhook 验签上下文。
type Webhook struct {
	Secret    string        // endpoint signing secret(whsec_...)
	Tolerance time.Duration // 时间戳容差(默认 5min,0 值同默认)
}

// DefaultTolerance 官方建议容差 5 分钟。
const DefaultTolerance = 5 * time.Minute

var (
	ErrBadSignature = errors.New("stripe: webhook signature mismatch")
	ErrStaleTS      = errors.New("stripe: webhook timestamp outside tolerance")
)

// Verify 校验 Stripe-Signature 头(t=...,v1=...):对 "t.<payload>" 做 HMAC-SHA256。
func (w Webhook) Verify(payload []byte, sigHeader string) error {
	ts, sigs := parseSigHeader(sigHeader)
	if ts == "" || len(sigs) == 0 {
		return ErrBadSignature
	}
	tol := w.Tolerance
	if tol == 0 {
		tol = DefaultTolerance
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || time.Since(time.Unix(sec, 0)) > tol {
		return ErrStaleTS
	}
	mac := hmac.New(sha256.New, []byte(w.Secret))
	mac.Write([]byte(ts + "."))
	mac.Write(payload)
	want := hex.EncodeToString(mac.Sum(nil))
	for _, s := range sigs {
		if subtle.ConstantTimeCompare([]byte(s), []byte(want)) == 1 {
			return nil
		}
	}
	return ErrBadSignature
}

// parseSigHeader 拆 t 与所有 v1 值;格式非法返回空。
func parseSigHeader(h string) (ts string, v1 []string) {
	for _, part := range strings.Split(h, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			ts = v
		case "v1":
			v1 = append(v1, v)
		}
	}
	return ts, v1
}

// Event 解析后的 webhook 事件(仅取收款链路所需字段)。
type Event struct {
	Type        string // payment_intent.succeeded / payment_intent.payment_failed / ...
	PayNo       string // metadata.pay_no(发起时回传)
	IntentID    string
	AmountCents int64
	Currency    string
	BillNo      string // metadata.bill_no(缴费意图携带,充值为空)
	CustomerID  int64  // metadata.customer_id
}

// ParseEvent 解析事件 JSON;metadata 以 []byte 兜底避免 null 解码报错。
func ParseEvent(payload []byte) (Event, error) {
	var raw struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID       string          `json:"id"`
				Amount   int64           `json:"amount"`
				Currency string          `json:"currency"`
				Metadata json.RawMessage `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Event{}, fmt.Errorf("stripe: decode event: %w", err)
	}
	meta := map[string]string{}
	if len(raw.Data.Object.Metadata) > 0 {
		_ = json.Unmarshal(raw.Data.Object.Metadata, &meta)
	}
	cid, _ := strconv.ParseInt(meta["customer_id"], 10, 64)
	return Event{
		Type: raw.Type, IntentID: raw.Data.Object.ID,
		AmountCents: raw.Data.Object.Amount, Currency: raw.Data.Object.Currency,
		PayNo: meta["pay_no"], BillNo: meta["bill_no"], CustomerID: cid,
	}, nil
}

// SignPayload 生成签名头(测试与本地联调用,与 Verify 互逆)。
func (w Webhook) SignPayload(payload []byte, at time.Time) string {
	ts := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(w.Secret))
	mac.Write([]byte(ts + "."))
	mac.Write(payload)
	return "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}
