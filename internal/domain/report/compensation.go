package report

// Q2 补偿任务中心:跨域可回放补偿任务的统一只读视图。
// 六类任务(下发/停复机/激活回调/税局发票/话单 Kafka/缴费对账批次),
// 每项给出寻址主键与可回放 admin 路径;话单为自动补偿无单发重试口。

import (
	"context"
	"fmt"
	"strconv"
)

// CompTask 补偿任务条目(跨域聚合,只读)。
type CompTask struct {
	Domain    string `json:"domain"` // provision/billing/order/aaa
	Type      string `json:"type"`   // provisionTask/stopResume/activationCallback/taxInvoice/cdrKafka/reconBatch
	RefID     string `json:"refId"`  // 寻址主键(taskNo/taskId/callbackId/invoiceNo/batchNo;聚合项为空)
	Status    string `json:"status"` // FAILED/DIFF_PENDING
	Detail    string `json:"detail"`
	RetryPath string `json:"retryPath"` // 可回放 admin POST;空=自动补偿或无单发口
}

// compLimit 单类上限(中心视图非分页清单,防大表刷屏)。
const compLimit = 50

// compQueries 六类补偿任务查询:均返回单列寻址主键(cdrKafka 为聚合计数)。
var compQueries = []struct {
	domain, typ, status, detail, retryFmt, sql string
}{
	{
		"provision", "provisionTask", "FAILED",
		"配置下发失败(装维零手工链路)",
		"/api/admin/v1/provision-tasks/%s/retry",
		`SELECT task_no FROM provision_tasks WHERE status = 'FAILED' ORDER BY id LIMIT ` + strconv.Itoa(compLimit),
	},
	{
		"billing", "stopResume", "FAILED",
		"停复机任务失败(LO 账号状态迁移未生效)",
		"/api/admin/v1/stop-resume-tasks/%s/retry",
		`SELECT id::text FROM stop_resume_tasks WHERE status = 'FAILED' ORDER BY id LIMIT ` + strconv.Itoa(compLimit),
	},
	{
		"order", "activationCallback", "FAILED",
		"订单激活回调失败(环节11 未落账)",
		"/api/admin/v1/activation-callbacks/%s/retry",
		`SELECT id::text FROM activation_callbacks WHERE result = 'FAILED' ORDER BY id LIMIT ` + strconv.Itoa(compLimit),
	},
	{
		"billing", "taxInvoice", "FAILED",
		"税局开具失败(tax_fail_reason 留痕;重跑出账即重试)",
		"",
		`SELECT invoice_no FROM invoices WHERE tax_status = 'FAILED' ORDER BY id LIMIT ` + strconv.Itoa(compLimit),
	},
	{
		"aaa", "cdrKafka", "FAILED",
		"话单 Kafka 补投失败(补偿循环每分钟自动重试,无需人工回放)",
		"",
		`SELECT count(*)::text FROM cdrs WHERE kafka_status = 'FAILED'`,
	},
	{
		"billing", "reconBatch", "DIFF_PENDING",
		"缴费对账差异挂起(需人工核对后平账)",
		"/api/admin/v1/reconciliations/%s/settle",
		`SELECT batch_no FROM reconciliation_batches WHERE status = 'DIFF_PENDING' ORDER BY id DESC LIMIT ` + strconv.Itoa(compLimit),
	},
}

// CompensationTasks 跨域聚合补偿任务清单(只读,单项失败即整轮报错)。
// cdrKafka 为聚合计数:0 不出条目,非 0 出一条汇总(RefID 空,计数进 Detail)。
func (s *PGStore) CompensationTasks(ctx context.Context) ([]CompTask, error) {
	out := make([]CompTask, 0, len(compQueries)*4)
	for _, q := range compQueries {
		rows, err := s.db.Query(ctx, q.sql)
		if err != nil {
			return nil, fmt.Errorf("report: comp %s.%s: %w", q.domain, q.typ, err)
		}
		for rows.Next() {
			var ref string
			if err := rows.Scan(&ref); err != nil {
				rows.Close()
				return nil, fmt.Errorf("report: comp scan %s.%s: %w", q.domain, q.typ, err)
			}
			t := CompTask{Domain: q.domain, Type: q.typ, RefID: ref,
				Status: q.status, Detail: q.detail}
			if q.typ == "cdrKafka" {
				if ref == "0" {
					continue
				}
				t.RefID = ""
				t.Detail = q.detail + ";当前失败 " + ref + " 条"
				out = append(out, t)
				continue
			}
			if q.retryFmt != "" {
				t.RetryPath = fmt.Sprintf(q.retryFmt, ref)
			}
			out = append(out, t)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("report: comp rows %s.%s: %w", q.domain, q.typ, err)
		}
	}
	return out, nil
}

// CompTaskLister Store 可选能力:跨域补偿任务聚合。
type CompTaskLister interface {
	CompensationTasks(ctx context.Context) ([]CompTask, error)
}

// ErrCompUnsupported Store 不支持补偿任务聚合(如测试 fake)。
var ErrCompUnsupported = fmt.Errorf("report: store does not support compensation tasks")

// CompensationTasks 补偿任务中心清单(只读跨域聚合)。
func (r *ReportService) CompensationTasks(ctx context.Context) ([]CompTask, error) {
	l, ok := r.St.(CompTaskLister)
	if !ok {
		return nil, ErrCompUnsupported
	}
	return l.CompensationTasks(ctx)
}
