package aaa

// per-NAS 注册表 PG 实现:协议面 LookupNas(解密密钥,fail-closed)+ 管理面 CRUD(加密落库)。
// 解密失败/编解码器缺失输出 [aaa] 可 grep 告警并拒绝,禁止静默降级。

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const nasCols = "id, name, nas_ip, vendor, coa_port, enabled, created_at, updated_at"

// LookupNas 按来源 IP 查注册表并解密密钥;未注册/停用/解密失败返回域错误(fail-closed)。
func (s *PGStore) LookupNas(ctx context.Context, ip string) (*NasAuth, error) {
	var (
		c         NasClient
		secretEnc string
	)
	err := s.db.QueryRow(ctx,
		"SELECT "+nasCols+", secret_enc FROM aaa_nas_clients WHERE nas_ip = $1", ip).
		Scan(&c.ID, &c.Name, &c.NasIP, &c.Vendor, &c.CoAPort, &c.Enabled, &c.CreatedAt, &c.UpdatedAt, &secretEnc)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: ip=%s", ErrNasNotFound, ip)
	}
	if err != nil {
		return nil, fmt.Errorf("aaa: nas lookup %s: %w", ip, err)
	}
	if !c.Enabled {
		return nil, fmt.Errorf("%w: name=%s ip=%s", ErrNasDisabled, c.Name, ip)
	}
	plain, err := s.decodeNasSecret(ip, c.Name, secretEnc)
	if err != nil {
		return nil, err
	}
	return &NasAuth{Client: c, Secret: []byte(plain)}, nil
}

// decodeNasSecret 解密 NAS 共享密钥;编解码器缺失/密文损坏一律留痕并拒绝。
func (s *PGStore) decodeNasSecret(ip, name, secretEnc string) (string, error) {
	if s.cred == nil {
		log.Printf("[aaa] NAS SECRET DECRYPT FAILED ip=%s: credential codec not configured", ip)
		return "", errors.New("aaa: credential codec not configured")
	}
	plain, err := s.cred.Decode(secretEnc)
	if err != nil {
		log.Printf("[aaa] NAS SECRET DECRYPT FAILED ip=%s name=%s: %v", ip, name, err)
		return "", fmt.Errorf("aaa: nas secret decrypt %s: %w", ip, err)
	}
	return plain, nil
}

// CreateNas 新建 NAS 客户端(密钥加密落库);IP 重复返回 ErrNasDuplicate。
func (s *PGStore) CreateNas(ctx context.Context, u NasUpsert) (int64, error) {
	u, enc, err := s.prepareNasWrite(u, true)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRow(ctx,
		"INSERT INTO aaa_nas_clients(name, nas_ip, secret_enc, vendor, coa_port, enabled) "+
			"VALUES($1,$2,$3,$4,$5,$6) RETURNING id",
		u.Name, u.NasIP, enc, string(u.Vendor), u.CoAPort, derefNasBool(u.Enabled, true)).Scan(&id)
	if isNasUniqueViolation(err) {
		return 0, fmt.Errorf("%w: ip=%s", ErrNasDuplicate, u.NasIP)
	}
	if err != nil {
		return 0, fmt.Errorf("aaa: create nas: %w", err)
	}
	return id, nil
}

// UpdateNas 更新 NAS 客户端;SecretPlain 空=不改密钥;不存在返回 ErrNotFound。
func (s *PGStore) UpdateNas(ctx context.Context, id int64, u NasUpsert) error {
	u, enc, err := s.prepareNasWrite(u, false)
	if err != nil {
		return err
	}
	var tag pgconn.CommandTag
	if enc == "" {
		tag, err = s.db.Exec(ctx,
			"UPDATE aaa_nas_clients SET name=$2, nas_ip=$3, vendor=$4, coa_port=$5, enabled=$6, "+
				"updated_at=now() WHERE id=$1",
			id, u.Name, u.NasIP, string(u.Vendor), u.CoAPort, derefNasBool(u.Enabled, true))
	} else {
		tag, err = s.db.Exec(ctx,
			"UPDATE aaa_nas_clients SET name=$2, nas_ip=$3, vendor=$4, coa_port=$5, enabled=$6, "+
				"secret_enc=$7, updated_at=now() WHERE id=$1",
			id, u.Name, u.NasIP, string(u.Vendor), u.CoAPort, derefNasBool(u.Enabled, true), enc)
	}
	if err != nil {
		if isNasUniqueViolation(err) {
			return fmt.Errorf("%w: ip=%s", ErrNasDuplicate, u.NasIP)
		}
		return fmt.Errorf("aaa: update nas %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteNas 删除 NAS 客户端;不存在返回 ErrNotFound。
func (s *PGStore) DeleteNas(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, "DELETE FROM aaa_nas_clients WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("aaa: delete nas %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetNas 按 id 查安全字段(无密钥);不存在返回 ErrNotFound。
func (s *PGStore) GetNas(ctx context.Context, id int64) (*NasClient, error) {
	var c NasClient
	err := s.db.QueryRow(ctx, "SELECT "+nasCols+" FROM aaa_nas_clients WHERE id = $1", id).
		Scan(&c.ID, &c.Name, &c.NasIP, &c.Vendor, &c.CoAPort, &c.Enabled, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("aaa: get nas %d: %w", id, err)
	}
	return &c, nil
}

// ListNasPage 管理端分页(keyword 模糊匹配名称/IP,vendor/enabled 过滤)。
func (s *PGStore) ListNasPage(ctx context.Context, q NasPage) (AdminPageResult[NasClient], error) {
	q = normalizeNasPage(q)
	where, args := nasWhere(q.Keyword, q.Vendor, q.Enabled)
	count, err := s.count(ctx, "aaa_nas_clients", where, args)
	if err != nil {
		return AdminPageResult[NasClient]{}, err
	}
	rows, err := s.db.Query(ctx,
		"SELECT "+nasCols+" FROM aaa_nas_clients WHERE "+where+
			" ORDER BY id LIMIT $"+fmt.Sprint(len(args)+1)+" OFFSET $"+fmt.Sprint(len(args)+2),
		append(args, q.PageSize, (q.Page-1)*q.PageSize)...) //nolint:gocritic // 与既有分页读法一致
	if err != nil {
		return AdminPageResult[NasClient]{}, fmt.Errorf("aaa: list nas page: %w", err)
	}
	defer rows.Close()
	items := make([]NasClient, 0, q.PageSize)
	for rows.Next() {
		var c NasClient
		if err := rows.Scan(&c.ID, &c.Name, &c.NasIP, &c.Vendor, &c.CoAPort, &c.Enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return AdminPageResult[NasClient]{}, fmt.Errorf("aaa: scan nas: %w", err)
		}
		items = append(items, c)
	}
	return AdminPageResult[NasClient]{Items: items, Total: count, Page: q.Page, PageSize: q.PageSize}, rows.Err()
}

// normalizeNasPage 分页参数归一(默认 20,上限 100)。
func normalizeNasPage(q NasPage) NasPage {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	return q
}

// nasWhere 组装过滤条件(keyword 名称/IP 模糊,vendor/enabled 精确)。
func nasWhere(keyword, vendor, enabled string) (string, []any) {
	where := "TRUE"
	var args []any
	if kw := strings.TrimSpace(keyword); kw != "" {
		args = append(args, "%"+kw+"%")
		where += fmt.Sprintf(" AND (name ILIKE $%d OR nas_ip ILIKE $%d)", len(args), len(args))
	}
	if vendor != "" {
		args = append(args, vendor)
		where += fmt.Sprintf(" AND vendor = $%d", len(args))
	}
	if enabled == "true" || enabled == "false" {
		args = append(args, enabled == "true")
		where += fmt.Sprintf(" AND enabled = $%d", len(args))
	}
	return where, args
}

// prepareNasWrite 归一载荷并按需编码密钥;create=true 时密钥必填,更新空=不改密钥。
func (s *PGStore) prepareNasWrite(u NasUpsert, create bool) (NasUpsert, string, error) {
	u.Name = strings.TrimSpace(u.Name)
	u.NasIP = strings.TrimSpace(u.NasIP)
	if u.Name == "" || u.NasIP == "" {
		return u, "", errors.New("aaa: nas name and ip required")
	}
	if u.Vendor == "" {
		u.Vendor = NasVendorGeneric
	}
	switch u.Vendor {
	case NasVendorHuawei, NasVendorZTE, NasVendorGeneric:
	default:
		return u, "", fmt.Errorf("aaa: nas vendor %q invalid", u.Vendor)
	}
	if u.CoAPort <= 0 {
		u.CoAPort = DefaultNasCoAPort
	}
	if u.SecretPlain == "" {
		if create {
			return u, "", errors.New("aaa: nas secret required")
		}
		return u, "", nil
	}
	if s.cred == nil {
		return u, "", errors.New("aaa: credential codec not configured")
	}
	enc, err := s.cred.Encode(u.SecretPlain)
	if err != nil {
		return u, "", fmt.Errorf("aaa: encode nas secret: %w", err)
	}
	return u, enc, nil
}

func derefNasBool(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func isNasUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
