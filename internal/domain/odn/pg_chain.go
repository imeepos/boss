package odn

// W3 资源链导入存储:逐行展开入库(导入域箱体设备 upsert + 链行指纹去重)。
// 失败路径红线:行级失败输出 [odn-import] 可 grep 日志并附行号/批次/原因,禁止静默吞错。

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// ChainImportResult 批次导入结果(逐行成功/失败与原因、行号定位)。
type ChainImportResult struct {
	BatchNo   string           `json:"batchNo"`
	Total     int              `json:"total"`
	Imported  int              `json:"imported"`
	Failed    int              `json:"failed"`
	Skipped   int              `json:"skipped"`
	Duplicate int              `json:"duplicate"`
	Rows      []ChainRowResult `json:"rows"`
}

// ChainRowResult 逐行结果。
type ChainRowResult struct {
	RowNo  int    `json:"rowNo"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ImportResourceChains 批次导入:②示例行过滤 → ①③④⑤行校验 → ⑥指纹去重 → 展开。
// 行级失败不中断批次;每行独立事务,展开失败该行整体回滚。
func (s *PGStore) ImportResourceChains(ctx context.Context, in ChainImportInput) (*ChainImportResult, error) {
	batchNo := fmt.Sprintf("odn-%d", time.Now().UnixNano())
	res := &ChainImportResult{BatchNo: batchNo, Total: len(in.Rows), Rows: []ChainRowResult{}}
	for i, raw := range in.Rows {
		lineNo := raw.RowNo
		if lineNo <= 0 {
			lineNo = i + 2 // 表头占模板第 1 行
		}
		out := ChainRowResult{RowNo: lineNo, Status: ChainRowImported}
		if i < in.ExampleRowCount {
			out.Status, out.Reason = ChainRowSkippedExample, "说明页示例行,过滤不导入"
		} else {
			out = s.importValidatedChainRow(ctx, batchNo, raw, lineNo)
		}
		res.Rows = append(res.Rows, out)
		switch out.Status {
		case ChainRowImported:
			res.Imported++
		case ChainRowFailed:
			res.Failed++
		case ChainRowSkippedExample:
			res.Skipped++
		case ChainRowDuplicate:
			res.Duplicate++
		}
	}
	return res, nil
}

// importValidatedChainRow 单行导入:校验拒绝→failed;指纹冲突→duplicate;库错→failed+日志。
func (s *PGStore) importValidatedChainRow(ctx context.Context, batchNo string, raw ResourceChainRow, lineNo int) ChainRowResult {
	rec, reason := ValidateChainRow(raw, lineNo)
	if reason != "" {
		log.Printf("[odn-import] ROW %d FAILED batch=%s reason=%s", lineNo, batchNo, reason)
		return ChainRowResult{RowNo: lineNo, Status: ChainRowFailed, Reason: reason}
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-import] ROW %d BEGIN FAILED batch=%s reason=%v", lineNo, batchNo, err)
		return ChainRowResult{RowNo: lineNo, Status: ChainRowFailed, Reason: "事务启动失败"}
	}
	if err := expandChainRow(ctx, tx, rec); err != nil {
		_ = tx.Rollback(ctx)
		log.Printf("[odn-import] ROW %d EXPAND FAILED batch=%s reason=%v", lineNo, batchNo, err)
		return ChainRowResult{RowNo: lineNo, Status: ChainRowFailed, Reason: "展开失败: " + err.Error()}
	}
	inserted, err := insertChainLine(ctx, tx, batchNo, rec)
	if err != nil {
		_ = tx.Rollback(ctx)
		log.Printf("[odn-import] ROW %d INSERT FAILED batch=%s reason=%v", lineNo, batchNo, err)
		return ChainRowResult{RowNo: lineNo, Status: ChainRowFailed, Reason: "入库失败: " + err.Error()}
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-import] ROW %d COMMIT FAILED batch=%s reason=%v", lineNo, batchNo, err)
		return ChainRowResult{RowNo: lineNo, Status: ChainRowFailed, Reason: "提交失败"}
	}
	if !inserted {
		return ChainRowResult{RowNo: lineNo, Status: ChainRowDuplicate, Reason: "重复资源关系,已去重"}
	}
	return ChainRowResult{RowNo: lineNo, Status: ChainRowImported}
}

// expandChainRow 展开为系统关系数据:链上出现的箱体编码逐一取/建导入域设备(无城市),
// 父级用同链已解析设备(层级边=说明页顺序);非规范格式编码(如站点前缀 ODF 引用)不展开,不猜填。
func expandChainRow(ctx context.Context, tx pgx.Tx, rec *ChainRecord) error {
	ids := map[string]int64{}
	steps := []struct {
		code, kind, parent string
	}{
		{rec.OccCode, DevOCC, ""},
		{rec.OdfCode, DevODF, ""},
		{rec.OdbCode, DevODB, DevOCC},
		{rec.ObdCode, DevOBD, DevODB},
		{rec.SdbCode, DevSDB, DevODB},
		{rec.SbdCode, DevSBD, DevSDB},
	}
	for _, st := range steps {
		if st.code == "" || !isChainDeviceCode(st.code) {
			continue
		}
		id, err := upsertChainDevice(ctx, tx, st.code, st.kind, rec.Lifecycle, ids[st.parent])
		if err != nil {
			return fmt.Errorf("odn: expand %s: %w", st.code, err)
		}
		ids[st.kind] = id
	}
	return nil
}

// isChainDeviceCode 编码符合规范设备格式才展开(站点前缀引用如 SITE001_ODF001_A 不展开)。
func isChainDeviceCode(code string) bool {
	_, err := ValidateDeviceCode(code)
	return err == nil
}

// upsertChainDevice 导入域设备取/建(无城市,部分唯一索引 uq_odn_device_box 分域);
// 既有行不动(lifecycle 不降级,已流转状态保持,存量语义不变)。
func upsertChainDevice(ctx context.Context, tx pgx.Tx, code, kind, lifecycle string, parentID int64) (int64, error) {
	var id int64
	lookup := "SELECT id FROM odn_device WHERE code=$1 AND kind=$2 AND prv_code IS NULL"
	err := tx.QueryRow(ctx, lookup+" AND status <> 'RETIRED' ORDER BY id LIMIT 1", code, kind).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("odn: lookup chain device: %w", err)
	}
	ins := "INSERT INTO odn_device (code, kind, parent_id, lifecycle_status) VALUES ($1,$2,NULLIF($3,0),$4)"
	if _, err := tx.Exec(ctx, ins+" ON CONFLICT DO NOTHING", code, kind, parentID, lifecycle); err != nil {
		return 0, fmt.Errorf("odn: insert chain device: %w", err)
	}
	err = tx.QueryRow(ctx, lookup+" ORDER BY id LIMIT 1", code, kind).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("odn: reload chain device: %w", err)
	}
	return id, nil
}

// insertChainLine 链行入库(⑥指纹唯一,冲突即重复行);③留空字段一律 NULL,无默认值推断。
func insertChainLine(ctx context.Context, tx pgx.Tx, batchNo string, rec *ChainRecord) (bool, error) {
	tag, err := tx.Exec(ctx, "INSERT INTO odn_resource_chain (batch_no, line_no, lifecycle_status, site_code, site_name, olt_code, odf_code, odf_port, occ_code, odb_code, obd_code, split1_ratio, split1_port, sdb_code, sbd_code, split2_ratio, split2_port, total_split, fiber_code, fr_to, port_status, laying_method, row_status, pece_status, remark, fingerprint) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26) ON CONFLICT (fingerprint) DO NOTHING",
		batchNo, rec.LineNo, rec.Lifecycle, rec.SiteCode, rec.SiteName, rec.OltCode,
		nullIfEmpty(rec.OdfCode), nullIfEmpty(rec.OdfPort), nullIfEmpty(rec.OccCode),
		nullIfEmpty(rec.OdbCode), nullIfEmpty(rec.ObdCode), nullInt(rec.Split1), nullIfEmpty(rec.Split1Port),
		nullIfEmpty(rec.SdbCode), nullIfEmpty(rec.SbdCode), nullInt(rec.Split2), nullIfEmpty(rec.Split2Port),
		nullInt(rec.TotalSplit), nullIfEmpty(rec.FiberCode), nullIfEmpty(rec.FrTo),
		nullIfEmpty(rec.PortStatus), nullIfEmpty(rec.LayingMethod), nullIfEmpty(rec.RowStatus),
		nullIfEmpty(rec.PeceStatus), nullIfEmpty(rec.Remark), rec.Fingerprint)
	if err != nil {
		return false, fmt.Errorf("odn: insert chain line: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// nullIfEmpty 留空=不入库该字段(说明页原则③)。
func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// nullInt 0=不入库。
func nullInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}
