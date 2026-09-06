// 资产/标签列表服务端分页查询(P3-T1):offset/limit + 条件过滤 + 排序白名单。
// 排序白名单是防注入硬边界:查询参数只允许白名单键,映射到表内真实列名;
// ORDER BY 恒带 id DESC tie-breaker,同值行序稳定,翻页不重不漏。
// limit 钳制(默认 50 上限 200)不报错;白名单外 sort 由 adminapi 层回 400,此处兜底 ErrInvalidSort。
package asset

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidSort 排序字段不在白名单(adminapi 层先行 400,此处兜底)。
var ErrInvalidSort = errors.New("asset: invalid sort field")

// 列表排序白名单:查询参数 → SQL 列名(键即对外契约,值必须是表内真实列)。
var (
	// AssetSortColumns assets 列表排序白名单(P3-T1)。
	AssetSortColumns = map[string]string{
		"created_at": "created_at",
		"asset_code": "asset_code",
		"status":     "status",
	}
	// TagSortColumns tags 列表排序白名单(P3-T1)。
	TagSortColumns = map[string]string{
		"created_at": "created_at",
		"tag_no":     "tag_no",
		"status":     "status",
	}
)

// 列表分页常量:缺省与钳制口径(P3-T1 契约,超限静默钳制不报错)。
const (
	ListDefaultLimit = 50
	ListMaxLimit     = 200
)

// ListQuery 列表分页查询参数。Sort 为白名单键(空=created_at);
// Q 按 asset_code/tag_no 前缀匹配;ModelID 0=不过滤。
type ListQuery struct {
	Offset  int
	Limit   int
	Status  string
	Type    string
	ModelID int64
	Q       string
	Sort    string
}

// AssetPage 资产列表页:items + total(过滤后总数)。
type AssetPage struct {
	Items []Asset
	Total int64
}

// TagPage 标签列表页:items + total(过滤后总数)。
type TagPage struct {
	Items []Tag
	Total int64
}

// clamp 分页归一:非正值回默认,超上限钳到上限(契约:钳制不报错)。
func (q ListQuery) clamp() ListQuery {
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Limit <= 0 {
		q.Limit = ListDefaultLimit
	}
	if q.Limit > ListMaxLimit {
		q.Limit = ListMaxLimit
	}
	return q
}

// sortCol 白名单校验并取 SQL 列名;空=created_at,越界 ErrInvalidSort。
func (q ListQuery) sortCol(cols map[string]string) (string, error) {
	key := q.Sort
	if key == "" {
		key = "created_at"
	}
	col, ok := cols[key]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrInvalidSort, key)
	}
	return col, nil
}

// likePrefix 前缀匹配防通配注入:转义 LIKE 特殊字符后拼右通配。
func likePrefix(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(q) + "%"
}

// cond 追加器:占位符序号随 args 增长,拼 WHERE 片段。
type condBuilder struct {
	conds []string
	args  []any
}

func (b *condBuilder) add(cond string, v any) {
	b.args = append(b.args, v)
	b.conds = append(b.conds, fmt.Sprintf(cond, len(b.args)))
}

func (b *condBuilder) where() (string, []any) {
	if len(b.conds) == 0 {
		return "", b.args
	}
	return "WHERE " + strings.Join(b.conds, " AND "), b.args
}

func assetConds(q ListQuery) *condBuilder {
	b := &condBuilder{}
	if q.Status != "" {
		b.add("status = $%d", q.Status)
	}
	if q.Type != "" {
		b.add("type = $%d", q.Type)
	}
	if q.ModelID > 0 {
		b.add("COALESCE(model_id, 0) = $%d", q.ModelID)
	}
	if q.Q != "" {
		b.add("asset_code LIKE $%d ESCAPE '\\'", likePrefix(q.Q))
	}
	return b
}

func tagConds(q ListQuery) *condBuilder {
	b := &condBuilder{}
	if q.Status != "" {
		b.add("status = $%d", q.Status)
	}
	if q.Q != "" {
		b.add("tag_no LIKE $%d ESCAPE '\\'", likePrefix(q.Q))
	}
	return b
}

// ListAssetsPage 资产分页列表:WHERE/ORDER/LIMIT 全参数化,排序列出自白名单映射。
func (s *PGStore) ListAssetsPage(ctx context.Context, q ListQuery) (*AssetPage, error) {
	q = q.clamp()
	col, err := q.sortCol(AssetSortColumns)
	if err != nil {
		return nil, err
	}
	b := assetConds(q)
	where, args := b.where()
	rows, err := s.db.Query(ctx, `
		SELECT id, asset_code, batch_id, legal_entity_id, legal_entity_name,
		       COALESCE(tag_id, 0), COALESCE(address_id, 0), COALESCE(region_id, 0),
		       COALESCE(region_name, ''), type, status, COALESCE(model_id, 0),
		       COALESCE(sn, ''), COALESCE(mac, ''), COALESCE(loid, '')
		FROM assets `+where+`
		ORDER BY `+col+` DESC, id DESC
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2),
		append(args, int64(q.Limit), int64(q.Offset))...)
	if err != nil {
		return nil, fmt.Errorf("asset: list assets page: %w", err)
	}
	defer rows.Close()
	page := &AssetPage{Items: make([]Asset, 0)}
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.AssetID, &a.AssetCode, &a.BatchID, &a.LegalEntityID, &a.LegalEntityName,
			&a.TagID, &a.AddressID, &a.RegionID, &a.RegionName, &a.Type, &a.Status, &a.ModelID,
			&a.SN, &a.MAC, &a.LOID); err != nil {
			return nil, fmt.Errorf("asset: scan asset page: %w", err)
		}
		page.Items = append(page.Items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("asset: iterate assets page: %w", err)
	}
	if err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM assets "+where, args...).Scan(&page.Total); err != nil {
		return nil, fmt.Errorf("asset: count assets page: %w", err)
	}
	return page, nil
}

// ListTagsPage 标签分页列表:口径同 ListAssetsPage(过滤只剩 status/q)。
func (s *PGStore) ListTagsPage(ctx context.Context, q ListQuery) (*TagPage, error) {
	q = q.clamp()
	col, err := q.sortCol(TagSortColumns)
	if err != nil {
		return nil, err
	}
	b := tagConds(q)
	where, args := b.where()
	rows, err := s.db.Query(ctx, `
		SELECT id, legal_entity_id, tag_no, epc_code, band, COALESCE(bound_asset_id, 0), status, battery
		FROM tags `+where+`
		ORDER BY `+col+` DESC, id DESC
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2),
		append(args, int64(q.Limit), int64(q.Offset))...)
	if err != nil {
		return nil, fmt.Errorf("asset: list tags page: %w", err)
	}
	defer rows.Close()
	page := &TagPage{Items: make([]Tag, 0)}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.TagID, &t.LegalEntityID, &t.TagNo, &t.EpcCode, &t.Band, &t.BoundAssetID, &t.Status, &t.Battery); err != nil {
			return nil, fmt.Errorf("asset: scan tag page: %w", err)
		}
		page.Items = append(page.Items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("asset: iterate tags page: %w", err)
	}
	if err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM tags "+where, args...).Scan(&page.Total); err != nil {
		return nil, fmt.Errorf("asset: count tags page: %w", err)
	}
	return page, nil
}
