package app

// Stripe webhook endpoint 自愈/告警循环:隧道快速 URL 变化后自动把 Stripe 后台
// endpoint 的 URL 同步到当前隧道,消除 webhook 静默失效;异常明确告警进后台提醒中心。
// 机制:期望 URL 来自 stripe.webhookUrl(配置页/biz_params,102 cron 在隧道变化时更新),
// 周期比对 Stripe 后台注册 endpoint:URL 失配则 UPDATE(签名密钥随 endpoint 不变,无需换
// whsec);无本系统 endpoint 则 CREATE 并落新 whsec 到 biz_params。见 adopted 2026-08-30。

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
	"github.com/ymm-001/boss/internal/pkg/stripe"
)

// stripeWebhookPath Stripe 回调路径(比对 endpoint URL 后缀)。
const stripeWebhookPath = "/api/user/v1/webhooks/stripe"

// stripeGuardInterval 自愈循环周期(静默失效窗口 ≈ 隧道发现 + 本周期)。
const stripeGuardInterval = 5 * time.Minute

// stripeGuardEvents 本系统订阅的收款事件。
var stripeGuardEvents = []string{"payment_intent.succeeded", "payment_intent.payment_failed"}

// stripeGuardParams 循环依赖的 biz_params 窄口(读写期望 URL / 持久化新 whsec)。
type stripeGuardParams interface {
	ListParams(ctx context.Context) ([]user.Param, error)
	UpdateParam(ctx context.Context, key, value string, updatedBy int64) error
}

// stripeGuardDeps 循环依赖最小集合(便于单测注入)。
type stripeGuardDeps struct {
	stripe *stripe.Dynamic
	params stripeGuardParams // 可空:无法持久化新 whsec 时仅告警
	n      notify.Service    // 可空:无提醒中心时仅记日志
	now    func() time.Time
}

// startStripeWebhookGuard 启动自愈循环,返回 stop(幂等)。
func startStripeWebhookGuard(a *Application) (stop func()) {
	if a.Stripe == nil || a.User == nil {
		return func() {}
	}
	d := stripeGuardDeps{stripe: a.Stripe}
	if ps, ok := a.User.(stripeGuardParams); ok {
		d.params = ps
	}
	d.n = a.Notify
	d.now = time.Now
	return runStripeWebhookGuard(d)
}

// runStripeWebhookGuard 周期执行一次自检自愈(模式同 reserve_timeout_loop)。
func runStripeWebhookGuard(d stripeGuardDeps) (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		runStripeWebhookOnce(ctx, d)
		t := time.NewTicker(stripeGuardInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runStripeWebhookOnce(ctx, d)
			}
		}
	}()
	return func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}
}

func runStripeWebhookOnce(ctx context.Context, d stripeGuardDeps) {
	cctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	client, err := d.stripe.Client(cctx)
	if err != nil {
		return // 未启用/缺凭据:通道本就没开,不自检不告警
	}
	cfg, err := d.stripe.Config(cctx)
	if err != nil || cfg.WebhookURL == "" {
		return // 期望 URL 未配置,无从比对(仅人工/ cron 配置后生效)
	}
	want := strings.TrimRight(cfg.WebhookURL, "/")
	endpoints, err := client.ListWebhookEndpoints(cctx)
	if err != nil {
		log.Printf("stripe webhook guard: list endpoints: %v", err)
		emitStripeWebhookAlert(cctx, d, "Stripe webhook 自检失败", "查询 webhook endpoint 失败: "+err.Error(), want)
		return
	}
	for _, ep := range endpoints {
		if !strings.Contains(ep.URL, stripeWebhookPath) {
			continue
		}
		if strings.TrimRight(ep.URL, "/") == want {
			return // 健康:URL 一致
		}
		// 隧道已变 → 自愈:更新 URL(签名密钥随 endpoint 不变)。
		if err := client.UpdateWebhookEndpoint(cctx, ep.ID, want); err != nil {
			log.Printf("stripe webhook guard: update %s: %v", ep.ID, err)
			emitStripeWebhookAlert(cctx, d, "Stripe webhook 自愈失败",
				"endpoint "+ep.ID+" URL 更新失败: "+err.Error(), want)
			return
		}
		log.Printf("stripe webhook guard: endpoint %s URL -> %s", ep.ID, want)
		emitStripeWebhookAlert(cctx, d, "Stripe webhook endpoint URL 已自愈",
			"endpoint "+ep.ID+" URL 已同步到 "+want, want)
		return
	}
	// 无本系统 endpoint → 重建(新 whsec 落 biz_params,60s 热生效)。
	ep, secret, err := client.CreateWebhookEndpoint(cctx, want, stripeGuardEvents)
	if err != nil {
		log.Printf("stripe webhook guard: create endpoint: %v", err)
		emitStripeWebhookAlert(cctx, d, "Stripe webhook endpoint 重建失败",
			"创建 endpoint 失败: "+err.Error(), want)
		return
	}
	if secret != "" && d.params != nil {
		if enc, err := secretbox.Seal(secret); err == nil {
			if err := d.params.UpdateParam(cctx, "stripe.webhookSecret", enc, 0); err != nil {
				log.Printf("stripe webhook guard: persist whsec: %v", err)
			}
		}
	}
	log.Printf("stripe webhook guard: endpoint %s created for %s", ep.ID, want)
	emitStripeWebhookAlert(cctx, d, "Stripe webhook endpoint 已重建",
		"endpoint "+ep.ID+" 已创建,新 whsec 已写入配置", want)
}

// emitStripeWebhookAlert 告警进后台提醒中心(RefID=期望 URL,同 URL 幂等防刷屏)。
func emitStripeWebhookAlert(ctx context.Context, d stripeGuardDeps, title, content, refID string) {
	if d.n == nil {
		return
	}
	if err := d.n.Emit(ctx, notify.Input{
		Category: notify.CategoryTask,
		Level:    notify.LevelWarn,
		Title:    title,
		Content:  content,
		Link:     "/base/stripeconfig",
		RefType:  "stripe_webhook_guard",
		RefID:    refID,
	}); err != nil {
		log.Printf("stripe webhook guard: emit: %v", err)
	}
}
