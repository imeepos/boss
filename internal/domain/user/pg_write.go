package user

// PGStore 写侧:自助改密/自助资料/地址批量导入(与 pg.go 读侧分文件,S4 整改)。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ChangePassword 自助改密:校验旧口令 → bcrypt 新口令 → 更新;旧口令错返回 ErrUnauthorized。
func (s *PGStore) ChangePassword(ctx context.Context, accountID int64, oldPassword, newPassword string) error {
	var hash string
	err := s.db.QueryRow(ctx, `SELECT password_hash FROM accounts WHERE id = $1`, accountID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUnauthorized
	}
	if err != nil {
		return fmt.Errorf("user: change password query: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword)) != nil {
		return ErrUnauthorized
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("user: bcrypt: %w", err)
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE accounts SET password_hash = $2, updated_at = now() WHERE id = $1`, accountID, string(newHash)); err != nil {
		return fmt.Errorf("user: change password update: %w", err)
	}
	return nil
}

// UpdateSelfProfile 自助改基本资料:仅 real_name/phone;空 phone 写 NULL。
func (s *PGStore) UpdateSelfProfile(ctx context.Context, accountID int64, realName, phone string) error {
	var ph *string
	if phone != "" {
		ph = &phone
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE accounts SET real_name = $2, phone = $3, updated_at = now() WHERE id = $1`,
		accountID, realName, ph)
	if err != nil {
		return fmt.Errorf("user: update self profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUnauthorized
	}
	return nil
}

// ImportAddresses 批量导入地址;level 与 parent_id 由 path 派生(应用层算,不手填)。
// 约束:子节点导入前父节点必须已存在(ltree 前缀父路径反查)。
func (s *PGStore) ImportAddresses(ctx context.Context, rows []AddressRow) (int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("user: import address begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	imported := 0
	for _, r := range rows {
		level := int8(len(strings.Split(r.Path, ".")))
		parentPath := parentOf(r.Path)
		var parentID int64
		if parentPath != "" {
			err := tx.QueryRow(ctx,
				`SELECT id FROM addresses WHERE path = $1::ltree`, parentPath).Scan(&parentID)
			if errors.Is(err, pgx.ErrNoRows) {
				return imported, fmt.Errorf("user: import address %q: parent %q not found", r.Path, parentPath)
			}
			if err != nil {
				return imported, fmt.Errorf("user: import address lookup parent: %w", err)
			}
		}
		var parentArg any
		if parentPath != "" {
			parentArg = parentID
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO addresses(path, level, name, parent_id, country_code, admin_code)
			VALUES($1::ltree, $2, $3, $4, NULLIF($5,''), NULLIF($6,''))`,
			r.Path, level, r.Name, parentArg, r.geoCountry(level), r.geoAdmin(level)); err != nil {
			if isPgCode(err, "23505") {
				return imported, ErrDuplicate
			}
			return imported, fmt.Errorf("user: import address insert: %w", err)
		}
		imported++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("user: import address commit: %w", err)
	}
	return imported, nil
}
