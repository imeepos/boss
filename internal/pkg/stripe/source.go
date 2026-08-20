package stripe

// 渠道侧流水拉取(对账自动化):按日拉 Balance Transactions,
// 经 expand data.source.payment_intent 回取 metadata.pay_no,与系统 payments.pay_no 对齐。
// 契约命名红线:渠道原始 snake_case 响应不经 struct tag,一律 map 取字段。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// StatementRow 对账用渠道流水行(金额单位元,对齐 billing.ChannelStatementRow)。
type StatementRow struct {
	ID     string // 渠道流水号(分页游标)
	PayNo  string // 空=无 pay_no 挂钩的收款(非本系统发起),对账侧按渠道独有处理
	Amount float64
}

// ListDayStatements 拉取某自然日(UTC)收款流水;分页取前 10 页(100/页)足够日批。
func (c *Client) ListDayStatements(ctx context.Context, day time.Time) ([]StatementRow, error) {
	start := time.Date(day.UTC().Year(), day.UTC().Month(), day.UTC().Day(), 0, 0, 0, 0, time.UTC)
	q := url.Values{}
	q.Set("limit", "100")
	q.Set("created[gte]", strconv.FormatInt(start.Unix(), 10))
	q.Set("created[lte]", strconv.FormatInt(start.Add(24*time.Hour).Unix()-1, 10))
	q.Set("expand[]", "data.source.payment_intent")

	rows := make([]StatementRow, 0)
	for page := 0; page < 10; page++ {
		if len(rows) > 0 {
			q.Set("starting_after", rows[len(rows)-1].ID)
		}
		batch, hasMore, err := c.fetchStatementPage(ctx, q)
		if err != nil {
			return nil, err
		}
		rows = append(rows, batch...)
		if !hasMore {
			break
		}
	}
	return rows, nil
}

// fetchStatementPage 单页拉取:仅取 charge(收款)行,退款/手续费不入比对。
func (c *Client) fetchStatementPage(ctx context.Context, q url.Values) ([]StatementRow, bool, error) {
	var raw map[string]any
	if err := c.getJSON(ctx, "/v1/balance_transactions?"+q.Encode(), &raw); err != nil {
		return nil, false, err
	}
	data, _ := raw["data"].([]any)
	rows := make([]StatementRow, 0, len(data))
	for _, item := range data {
		m, _ := item.(map[string]any)
		src, _ := m["source"].(map[string]any)
		if src == nil || src["object"] != "charge" {
			continue
		}
		pi, _ := src["payment_intent"].(map[string]any)
		meta, _ := pi["metadata"].(map[string]any)
		rows = append(rows, StatementRow{
			ID: toStr(m["id"]), PayNo: toStr(meta["pay_no"]), Amount: float64(toInt(m["amount"])) / 100,
		})
	}
	return rows, raw["has_more"] == true, nil
}

// getJSON GET 并解码;非 2xx 返回带状态码错误。
func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("stripe: get %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("stripe: %s status %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("stripe: decode %s: %w", path, err)
	}
	return nil
}
