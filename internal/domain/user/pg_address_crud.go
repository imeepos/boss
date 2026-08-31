package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrConflict 节点有子级或被业务表引用,不可删除。
var ErrConflict = errors.New("user: conflict")

// ErrDuplicate path 唯一键冲突(节点已存在)。
var ErrDuplicate = errors.New("user: duplicate")

// addrLabelRe path 段合法字符:小写字母/数字/下划线(ltree 标签约束)。
var addrLabelRe = regexp.MustCompile(`^[a-z0-9_]+$`)

// addressHitRow 搜索命中内部行(带 path 文本,用于派生祖先链)。
// addressHitRow 搜索命中内部行;Path 并入 Address.Path(服务层透出契约)。
type addressHitRow struct {
	Address
}

// CreateAddress 新增节点。parentID=0 建根(可带锚点);label 即 path 末段,path 拼接派生。
func (s *PGStore) CreateAddress(ctx context.Context, parentID int64,
	label, name, countryCode, adminCode string) (int64, error) {
	if !addrLabelRe.MatchString(label) {
		return 0, ErrInvalidInput
	}
	var (
		parentPath string
		level      = int8(1)
	)
	if parentID != 0 {
		err := s.db.QueryRow(ctx,
			`SELECT path::text, level FROM addresses WHERE id = $1`, parentID).
			Scan(&parentPath, &level)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		if err != nil {
			return 0, fmt.Errorf("user: create address lookup parent: %w", err)
		}
		if level >= 5 {
			return 0, ErrInvalidInput // 树深上限 5(000001 CHECK 同口径)
		}
		level++
	}
	if parentID != 0 && (countryCode != "" || adminCode != "") {
		return 0, ErrInvalidInput // 锚点只允许根节点,非根创建即拒
	}
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO addresses(path, level, name, parent_id, country_code, admin_code)
		VALUES($1::ltree, $2, $3, NULLIF($4,0), NULLIF($5,''), NULLIF($6,''))
		RETURNING id`,
		joinPath(parentPath, label), level, name, parentID, countryCode, adminCode).Scan(&id)
	if isPgCode(err, "23505") {
		return 0, ErrDuplicate
	}
	if err != nil {
		return 0, fmt.Errorf("user: create address: %w", err)
	}
	return id, nil
}

// joinPath 根节点 path=label 本身,子级=父 path.label。
func joinPath(parentPath, label string) string {
	if parentPath == "" {
		return label
	}
	return parentPath + "." + label
}

// SetAddressGeom 写入节点坐标(WGS84);ST_MakePoint 惯例 lng 在前,勿颠倒。
// 范围越界在应用层先拒(ErrInvalidInput),DB CHECK 兜底。
func (s *PGStore) SetAddressGeom(ctx context.Context, id int64, lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return ErrInvalidInput
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE addresses SET geom = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography WHERE id = $1`,
		id, lng, lat)
	if err != nil {
		return fmt.Errorf("user: set address geom: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateAddressName 仅改名;path 为权威不可变(ADR-002)。
func (s *PGStore) UpdateAddressName(ctx context.Context, id int64, name string) error {
	tag, err := s.db.Exec(ctx, `UPDATE addresses SET name = $2 WHERE id = $1`, id, name)
	if err != nil {
		return fmt.Errorf("user: update address name: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteAddress 删除叶节点;有子节点预检拒绝,被业务表 FK 引用映射 ErrConflict。
func (s *PGStore) DeleteAddress(ctx context.Context, id int64) error {
	var hasKids bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM addresses WHERE parent_id = $1)`, id).Scan(&hasKids)
	if err != nil {
		return fmt.Errorf("user: delete address check children: %w", err)
	}
	if hasKids {
		return ErrConflict
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM addresses WHERE id = $1`, id)
	if isPgCode(err, "23503") {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("user: delete address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SearchAddresses 全树关键字搜索(名称/path/锚点),带祖先链;单页上限 20 条,
// 多取 1 条探测截断返回 hasMore(此前静默截断,调用方无法感知结果不完整)。
func (s *PGStore) SearchAddresses(ctx context.Context, kw string) ([]AddressHit, bool, error) {
	like := "%" + kw + "%"
	rows, err := s.db.Query(ctx, `
		SELECT a.id, COALESCE(a.parent_id,0), a.level, a.name, a.path::text,
		       COALESCE(r.country_code,''), COALESCE(r.admin_code,''),
		       EXISTS(SELECT 1 FROM addresses c WHERE c.parent_id = a.id)
		FROM addresses a
		JOIN addresses r ON r.path = subpath(a.path, 0, 1)
		WHERE a.name ILIKE $1 OR a.path::text ILIKE $1
		   OR r.country_code ILIKE $1 OR r.admin_code ILIKE $1
		ORDER BY a.path LIMIT 21`, like)
	if err != nil {
		return nil, false, fmt.Errorf("user: search addresses: %w", err)
	}
	type hit = addressHitRow
	var hits []hit
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.ID, &h.ParentID, &h.Level, &h.Name, &h.Path,
			&h.CountryCode, &h.AdminCode, &h.HasChildren); err != nil {
			return nil, false, fmt.Errorf("user: scan search hit: %w", err)
		}
		hits = append(hits, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("user: search addresses rows: %w", err)
	}
	hasMore := len(hits) > 20
	if hasMore {
		hits = hits[:20]
	}
	if len(hits) == 0 {
		return []AddressHit{}, false, nil
	}
	out, err := s.attachAncestors(ctx, hits)
	return out, hasMore, err
}

// LookupAddresses 按 path 精确批量反查节点+祖先链;SQL 与 attachAncestors 反查同形状(ANY($1))。
// 命中按入参顺序返回(保序去重);缺失路径进 missing 不报错(address_path 弱引用,节点可删)。
func (s *PGStore) LookupAddresses(ctx context.Context, paths []string) ([]AddressHit, []string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT a.id, COALESCE(a.parent_id,0), a.level, a.name, a.path::text,
		       COALESCE(r.country_code,''), COALESCE(r.admin_code,''),
		       EXISTS(SELECT 1 FROM addresses c WHERE c.parent_id = a.id)
		FROM addresses a
		JOIN addresses r ON r.path = subpath(a.path, 0, 1)
		WHERE a.path::text = ANY($1)`, paths)
	if err != nil {
		return nil, nil, fmt.Errorf("user: lookup addresses: %w", err)
	}
	found := map[string]addressHitRow{}
	for rows.Next() {
		var h addressHitRow
		if err := rows.Scan(&h.ID, &h.ParentID, &h.Level, &h.Name, &h.Path,
			&h.CountryCode, &h.AdminCode, &h.HasChildren); err != nil {
			return nil, nil, fmt.Errorf("user: scan lookup hit: %w", err)
		}
		found[h.Path] = h
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("user: lookup addresses rows: %w", err)
	}
	ordered := make([]addressHitRow, 0, len(paths))
	missing := []string{}
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		if h, ok := found[p]; ok {
			ordered = append(ordered, h)
		} else {
			missing = append(missing, p)
		}
	}
	hits, err := s.attachAncestors(ctx, ordered)
	if err != nil {
		return nil, nil, err
	}
	return hits, missing, nil
}

// attachAncestors 批量反查命中节点全部祖先(path 前缀段),组装按 level 升序的祖先链。
// hasChildren 一并回填:此前祖先该字段恒 false(零值),客户端据其判断能否继续下钻会误判叶节点。
func (s *PGStore) attachAncestors(ctx context.Context, hits []addressHitRow) ([]AddressHit, error) {
	prefixes := map[string]bool{}
	for _, h := range hits {
		for _, p := range ancestorPaths(h.Path) {
			prefixes[p] = true
		}
	}
	paths := make([]string, 0, len(prefixes))
	for p := range prefixes {
		paths = append(paths, p)
	}
	rows, err := s.db.Query(ctx, `
		SELECT a.id, COALESCE(a.parent_id,0), a.level, a.name, a.path::text,
		       EXISTS(SELECT 1 FROM addresses c WHERE c.parent_id = a.id)
		FROM addresses a WHERE a.path::text = ANY($1) ORDER BY path`, paths)
	if err != nil {
		return nil, fmt.Errorf("user: search ancestors: %w", err)
	}
	byPath := map[string]Address{}
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.ParentID, &a.Level, &a.Name, &a.Path, &a.HasChildren); err != nil {
			return nil, fmt.Errorf("user: scan ancestor: %w", err)
		}
		byPath[a.Path] = a
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("user: search ancestors rows: %w", err)
	}
	out := make([]AddressHit, 0, len(hits))
	for _, h := range hits {
		chain := []Address{}
		for _, p := range ancestorPaths(h.Path) {
			if a, ok := byPath[p]; ok {
				chain = append(chain, a)
			}
		}
		out = append(out, AddressHit{Node: h.Address, Ancestors: chain})
	}
	return out, nil
}

// ancestorPaths path 的全部真祖先路径,按层级升序;根节点返回空。
func ancestorPaths(path string) []string {
	segs := strings.Split(path, ".")
	out := make([]string, 0, len(segs)-1)
	for i := 1; i < len(segs); i++ {
		out = append(out, strings.Join(segs[:i], "."))
	}
	return out
}

// isPgCode 判 pg 错误码(唯一/外联冲突)。
func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
