package monthly

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// tableSpec 三张对接 CSV 的列契约。表头与 docs/books/模板_月度数据_*.csv
// 逐字节一致(含单位后缀);dbCols 顺序=CSV 量值列顺序=INSERT 列顺序。
type tableSpec struct {
	kind   string   // 路由段/任务 kind
	table  string   // DB 表名
	header []string // CSV 表头(BOM 之外)
	dbCols []string // 量值 DB 列名(不含 month/region)
}

var tableSpecs = map[string]tableSpec{
	TableUserRevenue: {
		kind:  TableUserRevenue,
		table: "monthly_user_revenue",
		header: []string{"月份", "区域", "期初在用 (户)", "当月新增 (户)", "当月离网 (户)", "数据调整 (户)",
			"宽带收入 (₱)", "增值收入 (₱)", "一次性收费 (₱)", "优惠减免 (₱)", "退款冲销 (₱)"},
		dbCols: []string{"opening_active", "new_users", "churned_users", "adjusted_users",
			"broadband_revenue", "value_added_revenue", "onetime_charge", "discount_amount", "refund_reversal"},
	},
	TableNetworkDelivery: {
		kind:   TableNetworkDelivery,
		table:  "monthly_network_delivery",
		header: []string{"月份", "区域", "装机申请 (件)", "及时完工 (件)", "部署端口 (个)", "在用端口 (个)", "故障申报 (件)", "修复工时 (小时)"},
		dbCols: []string{"install_requests", "ontime_completions", "ports_deployed", "ports_active", "fault_reports", "repair_hours"},
	},
	TableFinanceCost: {
		kind:   TableFinanceCost,
		table:  "monthly_finance_cost",
		header: []string{"月份", "区域", "开票金额 (₱)", "实际回款 (₱)", "期末应收 (₱)", "直接成本 (₱)", "固定费用 (₱)", "CAPEX投入 (₱)"},
		dbCols: []string{"invoiced_amount", "collected_amount", "receivable_ending", "direct_cost", "fixed_cost", "capex_invest"},
	},
}

// specOf 表标识 → 列契约(未知标识 ErrUnknownTable)。
func specOf(kind string) (tableSpec, error) {
	spec, ok := tableSpecs[kind]
	if !ok {
		return tableSpec{}, ErrUnknownTable
	}
	return spec, nil
}

// bom UTF-8 BOM(权威输入为 UTF-8 with BOM,见模板说明【系统对接】)。
var bom = []byte{0xEF, 0xBB, 0xBF}

// parseCSVTable 解析上传 CSV:兼容 BOM;首行与模板表头逐字节比对;
// 返回数据行(1 基行号 = 数组下标 + 2,表头为第 1 行)。表头不符整体拒绝。
func parseCSVTable(kind string, data []byte) ([][]string, error) {
	spec, err := specOf(kind)
	if err != nil {
		return nil, err
	}
	body := bytes.TrimPrefix(data, bom)
	first, rest, err := splitFirstLine(body)
	if err != nil {
		return nil, fmt.Errorf("monthly: empty csv: %w", err)
	}
	want := strings.Join(spec.header, ",")
	if first != want {
		return nil, fmt.Errorf("monthly: %w: header mismatch: got %q want %q", ErrInvalidInput, first, want)
	}
	r := csv.NewReader(bytes.NewReader(rest))
	r.FieldsPerRecord = len(spec.header)
	var rows [][]string
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("monthly: csv row %d: %w", len(rows)+2, err)
		}
		rows = append(rows, rec)
	}
	return rows, nil
}

// splitFirstLine 切出首行(容 CRLF/LF);字节级比对表头用。
func splitFirstLine(b []byte) (string, []byte, error) {
	trimmed := bytes.TrimPrefix(b, []byte("\xEF\xBB\xBF"))
	i := bytes.IndexByte(trimmed, '\n')
	if i < 0 {
		return "", nil, errors.New("no line")
	}
	first := strings.TrimSuffix(string(trimmed[:i]), "\r")
	return first, trimmed[i+1:], nil
}

// csvRow 一条事实行:维度 + 量值(顺序=CSV 量值列)。
type csvRow struct {
	Month  string
	Region string
	Values []int64
}

// decodeRow 校验并解码一条数据行:行宽/月份格式/数值非负整数。
func decodeRow(kind string, rec []string) (csvRow, error) {
	spec, err := specOf(kind)
	if err != nil {
		return csvRow{}, err
	}
	if len(rec) != len(spec.header) {
		return csvRow{}, fmt.Errorf("列数 %d != %d", len(rec), len(spec.header))
	}
	if err := ValidateMonth(rec[0]); err != nil {
		return csvRow{}, fmt.Errorf("月份 %q 非 YYYY-MM", rec[0])
	}
	if rec[1] == "" {
		return csvRow{}, errors.New("区域为空")
	}
	vals := make([]int64, len(spec.dbCols))
	for i, raw := range rec[2:] {
		n, perr := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if perr != nil || n < 0 {
			return csvRow{}, fmt.Errorf("%s=%q 非非负整数", spec.header[i+2], raw)
		}
		vals[i] = n
	}
	return csvRow{Month: rec[0], Region: rec[1], Values: vals}, nil
}

// encode 导出行(数值十进制,无引号需求)。
func (r csvRow) encode(kind string) []string {
	out := make([]string, 0, len(r.Values)+2)
	out = append(out, r.Month, r.Region)
	for _, v := range r.Values {
		out = append(out, strconv.FormatInt(v, 10))
	}
	return out
}

// buildCSV 导出字节:BOM + CRLF 行尾 + 模板表头,与导入字节级闭环。
func buildCSV(kind string, rows []csvRow) ([]byte, error) {
	if _, err := specOf(kind); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Write(bom)
	w := csv.NewWriter(&buf)
	w.UseCRLF = true
	if err := w.Write(tableSpecs[kind].header); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if err := w.Write(r.encode(kind)); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
