package odn

// W3 资源链巡检与清单:导入完成后孤儿/半链检查(链上引用的资源必须存在)。

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// q SQL 字符串字面量(单引号包裹,免 Go 侧转义)。
func q(v string) string { return "'" + v + "'" }

// ChainPatrol 孤儿/半链巡检结果(计数 + 各类样本行号 ≤5)。
type ChainPatrol struct {
	Chains         int64              `json:"chains"`
	OrphanBoxCode  int64              `json:"orphanBoxCode"`
	BrokenAncestor int64              `json:"brokenAncestor"`
	SplitPortGap   int64              `json:"splitPortGap"`
	InfoRefMissing int64              `json:"infoRefMissing"`
	Samples        map[string][]int64 `json:"samples"`
}

// PatrolResourceChains 四类巡检:
// orphan_box_code=链上箱体编码无导入域设备行(孤儿);broken_ancestor=导入域设备缺父或父类型错(断祖);
// split_port_gap=有分光比无端口(半链);info_ref_missing=机房/OLT 引用为空(信息级,引用字段允许留空)。
func (s *PGStore) PatrolResourceChains(ctx context.Context) (*ChainPatrol, error) {
	p := &ChainPatrol{Samples: map[string][]int64{}}
	orphan := chainOrphanWhere()
	sql := "SELECT (SELECT count(*) FROM odn_resource_chain),"
	sql += " (SELECT count(*) FROM odn_resource_chain c WHERE" + orphan[4:] + "),"
	sql += " (SELECT count(*) FROM odn_device d WHERE d.prv_code IS NULL AND d.status <>" + q("RETIRED")
	sql += " AND d.kind IN (" + q("ODB") + "," + q("OBD") + "," + q("SDB") + "," + q("SBD") + ")"
	sql += " AND (d.parent_id IS NULL OR NOT EXISTS (SELECT 1 FROM odn_device p WHERE p.id=d.parent_id AND p.status <>" + q("RETIRED") + "))),"
	sql += " (SELECT count(*) FROM odn_resource_chain c WHERE (c.split1_ratio IS NOT NULL AND c.split1_port IS NULL) OR (c.split2_ratio IS NOT NULL AND c.split2_port IS NULL)),"
	sql += " (SELECT count(*) FROM odn_resource_chain c WHERE c.site_code=" + q("") + " OR c.olt_code=" + q("") + ")"
	if err := s.db.QueryRow(ctx, sql).Scan(&p.Chains, &p.OrphanBoxCode, &p.BrokenAncestor,
		&p.SplitPortGap, &p.InfoRefMissing); err != nil {
		return nil, fmt.Errorf("odn: patrol counts: %w", err)
	}
	samples := []struct{ key, where string }{
		{"orphan_box_code", orphan[4:]},
		{"split_port_gap", "(c.split1_ratio IS NOT NULL AND c.split1_port IS NULL) OR (c.split2_ratio IS NOT NULL AND c.split2_port IS NULL)"},
		{"info_ref_missing", "c.site_code=" + q("") + " OR c.olt_code=" + q("")},
	}
	for _, c := range samples {
		if err := s.patrolSample(ctx, p, c.key, c.where); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// chainOrphanWhere 箱体孤儿条件:链上箱体编码在导入域(无城市设备)无对应在用行。
func chainOrphanWhere() string {
	cols := []struct{ col, kind string }{
		{"occ_code", DevOCC}, {"odb_code", DevODB}, {"obd_code", DevOBD},
		{"sdb_code", DevSDB}, {"sbd_code", DevSBD},
	}
	orphan := ""
	for _, c := range cols {
		orphan += " OR (c." + c.col + " IS NOT NULL AND NOT EXISTS (SELECT 1 FROM odn_device d WHERE d.code=c." + c.col + " AND d.kind=" + q(c.kind) + " AND d.prv_code IS NULL AND d.status <>" + q("RETIRED") + "))"
	}
	return orphan
}

// patrolSample 采样违规行号 ≤5(样本表为 odn_resource_chain)。
func (s *PGStore) patrolSample(ctx context.Context, p *ChainPatrol, key, where string) error {
	rows, err := s.db.Query(ctx, "SELECT line_no FROM odn_resource_chain c WHERE "+where+" ORDER BY line_no LIMIT 5")
	if err != nil {
		return fmt.Errorf("odn: patrol sample %s: %w", key, err)
	}
	defer rows.Close()
	for rows.Next() {
		var n int64
		if err := rows.Scan(&n); err != nil {
			return fmt.Errorf("odn: patrol sample scan %s: %w", key, err)
		}
		p.Samples[key] = append(p.Samples[key], n)
	}
	return rows.Err()
}

// ResourceChainView 链行视图(清单/验收核对;枚举列为系统状态码,空串=未入库)。
type ResourceChainView struct {
	ID           int64  `json:"id"`
	BatchNo      string `json:"batchNo"`
	LineNo       int    `json:"lineNo"`
	Lifecycle    string `json:"lifecycleStatus"`
	SiteCode     string `json:"siteCode"`
	SiteName     string `json:"siteName"`
	OltCode      string `json:"oltCode"`
	OdfCode      string `json:"odfCode"`
	OdfPort      string `json:"odfPort"`
	OccCode      string `json:"occCode"`
	OdbCode      string `json:"odbCode"`
	ObdCode      string `json:"obdCode"`
	Split1       int    `json:"split1"`
	Split1Port   string `json:"split1Port"`
	SdbCode      string `json:"sdbCode"`
	SbdCode      string `json:"sbdCode"`
	Split2       int    `json:"split2"`
	Split2Port   string `json:"split2Port"`
	TotalSplit   int    `json:"totalSplit"`
	FiberCode    string `json:"fiberCode"`
	FrTo         string `json:"frTo"`
	PortStatus   string `json:"portStatus"`
	LayingMethod string `json:"layingMethod"`
	RowStatus    string `json:"rowStatus"`
	PeceStatus   string `json:"peceStatus"`
	Remark       string `json:"remark"`
	CreatedAt    string `json:"createdAt"`
}

// ListResourceChains 链行清单(batch 可空=全部;limit<=0 取 200)。
func (s *PGStore) ListResourceChains(ctx context.Context, batch string, limit int) ([]ResourceChainView, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	sql := "SELECT id, batch_no, line_no, lifecycle_status, site_code, site_name, olt_code, COALESCE(odf_code,''), COALESCE(odf_port,'')"
	sql += ", COALESCE(occ_code,''), COALESCE(odb_code,''), COALESCE(obd_code,''), COALESCE(split1_ratio,0), COALESCE(split1_port,''), COALESCE(sdb_code,''), COALESCE(sbd_code,'')"
	sql += ", COALESCE(split2_ratio,0), COALESCE(split2_port,''), COALESCE(total_split,0), COALESCE(fiber_code,''), COALESCE(fr_to,''), COALESCE(port_status,''), COALESCE(laying_method,'')"
	sql += ", COALESCE(row_status,''), COALESCE(pece_status,''), COALESCE(remark,''), created_at"
	sql += " FROM odn_resource_chain WHERE ($1=" + q("") + " OR batch_no=$1) ORDER BY id DESC"
	rows, err := s.db.Query(ctx, sql+" LIMIT "+fmt.Sprint(limit), batch)
	if err != nil {
		return nil, fmt.Errorf("odn: list chains: %w", err)
	}
	defer rows.Close()
	out := []ResourceChainView{}
	for rows.Next() {
		var v ResourceChainView
		var ts pgtype.Timestamptz
		if err := rows.Scan(&v.ID, &v.BatchNo, &v.LineNo, &v.Lifecycle, &v.SiteCode, &v.SiteName,
			&v.OltCode, &v.OdfCode, &v.OdfPort, &v.OccCode, &v.OdbCode, &v.ObdCode,
			&v.Split1, &v.Split1Port, &v.SdbCode, &v.SbdCode, &v.Split2, &v.Split2Port,
			&v.TotalSplit, &v.FiberCode, &v.FrTo, &v.PortStatus, &v.LayingMethod, &v.RowStatus,
			&v.PeceStatus, &v.Remark, &ts); err != nil {
			return nil, fmt.Errorf("odn: scan chain: %w", err)
		}
		if ts.Valid {
			v.CreatedAt = ts.Time.Format(time.RFC3339)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
