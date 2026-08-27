// WebhookDispatcher 的 Emit/DeliverDue 逻辑(存储经 WebhookStore 注入)。
package openplat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// EmitToApp 向指定应用的全部启用订阅广播事件(不按事件类型过滤):
// 用于 openplat.test 测试事件自检,集成方无需预先订阅该事件类型即可收到。
func (d *WebhookDispatcher) EmitToApp(ctx context.Context, appID int64, eventType, eventID string, payload any) (int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("openplat: emit marshal: %w", err)
	}
	n, err := d.store.InsertAppDeliveries(ctx, appID, eventType, eventID, body)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// DeliverDue 处理一批到期投递(ListDue 原子领取):逐条 POST(带 HMAC 签名头),按结果落库。
// MarkResult 失败必须留日志:结果写不回时行会在租约到期后重投,静默吞掉会造成
// 「同一事件反复重投且无迹可查」。返回 error 会中止同批后续投递,故仅记录不中断。
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
		switch {
		case postErr != nil:
			d.markLogged(ctx, dl.ID, false, 0, postErr.Error())
		case status >= 200 && status < 300:
			d.markLogged(ctx, dl.ID, true, status, "")
		default:
			d.markLogged(ctx, dl.ID, false, status, fmt.Sprintf("http %d", status))
		}
	}
	return len(due), nil
}

// markLogged 落投递结果;失败仅记日志(投递器循环下一轮按租约重新领取)。
func (d *WebhookDispatcher) markLogged(ctx context.Context, id int64, ok bool, status int, errMsg string) {
	if err := d.store.MarkResult(ctx, id, ok, status, errMsg); err != nil {
		log.Printf("[openplat-webhook] mark result id=%d ok=%v: %v", id, ok, err)
	}
}
