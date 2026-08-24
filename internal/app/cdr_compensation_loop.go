package app

// Q2 话单补偿:未送达 Kafka 的话单按轮补投。PG(cdrs)为权威侧,
// 实时链路失败置 FAILED/PENDING 留痕,本循环限速重放并回写 SENT;
// 失败留痕 + task 通知(每日幂等),支撑"关键补偿任务可回放"验收。

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaability "github.com/ymm-001/boss/internal/domain/aaa/billing"
	"github.com/ymm-001/boss/internal/domain/notify"
)

// cdrCompStore 补偿读口(PGStore 实现,窄口断言不污染 AaaService)。
type cdrCompStore interface {
	ListUnsentCdrs(ctx context.Context, limit int) ([]aaa.CdrRecord, error)
	MarkCdrsKafkaStatus(ctx context.Context, ids []int64, status string) error
}

// cdrCompDeps 补偿循环最小依赖,便于单测注入。
type cdrCompDeps struct {
	store   cdrCompStore      // 必备:权威侧读/回写
	emit    aaability.Emitter // 必备:Kafka 实时链路
	n       notify.Service    // 可空:失败留痕通知
	batch   int               // 每轮上限
	nowFunc func() time.Time  // 可替换时钟(测试)
}

const (
	cdrCompInterval = time.Minute
	cdrCompBatch    = 200
)

// startCdrCompensationLoop 启动话单补偿循环,返回 stop(幂等)。
// store 或实时链路未装配时空操作(未部署 Kafka 环境无补偿对象)。
func startCdrCompensationLoop(store cdrCompStore, emit aaability.Emitter, n notify.Service) (stop func()) {
	if store == nil || emit == nil {
		return func() {}
	}
	d := cdrCompDeps{store: store, emit: emit, n: n, batch: cdrCompBatch, nowFunc: time.Now}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		staggeredFirstRun(ctx, startupDelays["cdr_compensation"], func(c context.Context) { runCdrCompOnce(c, d) })
		t := time.NewTicker(cdrCompInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runCdrCompOnce(ctx, d)
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

// runCdrCompOnce 一轮:取未送达批 → 逐条补投 → 按结果回写 SENT/FAILED。
func runCdrCompOnce(ctx context.Context, d cdrCompDeps) {
	cctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cdrs, err := d.store.ListUnsentCdrs(cctx, d.batch)
	if err != nil {
		log.Printf("cdr compensation: list: %v", err)
		return
	}
	var sent, failed []int64
	for _, c := range cdrs {
		if err := d.emit.Emit(cctx, cdrRecordToCDR(c)); err != nil {
			failed = append(failed, c.ID)
			continue
		}
		sent = append(sent, c.ID)
	}
	if err := d.store.MarkCdrsKafkaStatus(cctx, sent, aaa.CdrKafkaSent); err != nil {
		log.Printf("cdr compensation: mark sent: %v", err)
	}
	if err := d.store.MarkCdrsKafkaStatus(cctx, failed, aaa.CdrKafkaFailed); err != nil {
		log.Printf("cdr compensation: mark failed: %v", err)
	}
	if len(failed) > 0 {
		emitCdrCompNotice(cctx, d, len(failed))
	}
}

// cdrRecordToCDR 权威侧记录 → 投递契约(Kafka value 与实时链路同构)。
func cdrRecordToCDR(c aaa.CdrRecord) aaability.CDR {
	return aaability.CDR{
		LOID:         c.Loid,
		Username:     c.Username,
		AcctStatus:   int(c.AcctStatus),
		SessionID:    c.SessionID,
		SessionTime:  uint32(c.SessionTime),
		InputOctets:  uint64(c.InputOctets),
		OutputOctets: uint64(c.OutputOctets),
		NASIP:        c.NasIP,
	}
}

// emitCdrCompNotice 补投失败留痕:task 通知,refID=当日(每日幂等聚合)。
func emitCdrCompNotice(ctx context.Context, d cdrCompDeps, failed int) {
	if d.n == nil {
		return
	}
	now := d.nowFunc()
	in := notify.Input{
		Category: notify.CategoryTask,
		Level:    notify.LevelWarn,
		Title:    "话单补偿存在失败",
		Content:  "Kafka 补投失败 " + strconv.Itoa(failed) + " 条,下一轮自动重试;持续失败请检查实时链路",
		Link:     "/aaa/aaalog",
		RefType:  "cdr_compensation",
		RefID:    now.Format("20060102"),
	}
	if err := d.n.Emit(ctx, in); err != nil {
		log.Printf("cdr compensation: emit: %v", err)
	}
}
