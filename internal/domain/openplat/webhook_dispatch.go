// WebhookDispatcher 的 Emit/DeliverDue 逻辑(存储经 WebhookStore 注入)。
package openplat

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Emit 广播事件:payload 序列化后按匹配订阅落 outbox(幂等)。
func (d *WebhookDispatcher) Emit(ctx context.Context, eventType, eventID string, payload any) (int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("openplat: emit marshal: %w", err)
	}
	n, err := d.store.InsertDeliveries(ctx, eventType, eventID, body)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// DeliverDue 处理一批到期投递:逐条 POST(带 HMAC 签名头),按结果落库。
func (d *WebhookDispatcher) DeliverDue(ctx context.Context) (int, error) {
	due, err := d.store.ListDue(ctx, time.Now(), DeliveryBatchMax)
	if err != nil {
		return 0, err
	}
	for _, dl := range due {
		body := []byte(dl.Payload)
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		headers := map[string]string{
			"Content-Type":     "application/json",
			"X-BOSS-Event":     dl.EventType,
			"X-BOSS-EventID":   dl.EventID,
			"X-BOSS-Timestamp": ts,
			"X-BOSS-Signature": "t=" + ts + ",v1=" + SignPayload(dl.Secret, ts, body),
		}
		status, postErr := d.poster.Post(dl.EndpointURL, headers, body)
		if postErr != nil {
			_ = d.store.MarkResult(ctx, dl.ID, false, 0, postErr.Error())
			continue
		}
		if status >= 200 && status < 300 {
			_ = d.store.MarkResult(ctx, dl.ID, true, status, "")
			continue
		}
		_ = d.store.MarkResult(ctx, dl.ID, false, status, fmt.Sprintf("http %d", status))
	}
	return len(due), nil
}
