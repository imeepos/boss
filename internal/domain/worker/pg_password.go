package worker

// 师傅登录密码管理 PG 实现(admin 录入 + 师傅端密码登录)。
// workers.password_hash 仅存 bcrypt 哈希不存明文(000043 列注释);空哈希不可密码登录。

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// isUniqueViolation 唯一约束冲突(workers.staff_no UNIQUE)。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CreateWorkerWithPassword 新建师傅并写入登录密码,返回自增 id。
// 校验 group_id/region_id 存在性,防止孤儿师傅;password 非空且长度 < PasswordMin 拒绝;
// 工号已占用返回 ErrDuplicate。
func (s *PGStore) CreateWorkerWithPassword(ctx context.Context, w Worker, password string) (int64, error) {
	if password != "" && utf8.RuneCountInString(password) < PasswordMin {
		return 0, ErrInvalidPassword
	}
	if w.GroupID > 0 {
		ok, err := s.exists(ctx, "worker_groups", w.GroupID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("worker: group %d: %w", w.GroupID, ErrForeignKeyViolation)
		}
	}
	if w.RegionID > 0 {
		ok, err := s.exists(ctx, "regions", w.RegionID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("worker: region %d: %w", w.RegionID, ErrForeignKeyViolation)
		}
	}
	hash := ""
	if password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return 0, fmt.Errorf("worker: bcrypt: %w", err)
		}
		hash = string(h)
	}
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO workers(staff_no, name, group_id, region_id, phone, status, joined_at, left_at, password_hash)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		w.StaffNo, w.Name, w.GroupID, w.RegionID, w.Phone, w.Status, w.JoinedAt, w.LeftAt, hash).Scan(&id)
	if isUniqueViolation(err) {
		return 0, ErrDuplicate
	}
	if err != nil {
		return 0, fmt.Errorf("worker: create worker: %w", err)
	}
	return id, nil
}

// SetPassword 重置师傅登录密码(bcrypt 落库);长度 < PasswordMin 返回 ErrInvalidPassword;
// 未命中返回 ErrNotFound。
func (s *PGStore) SetPassword(ctx context.Context, workerID int64, password string) error {
	if utf8.RuneCountInString(password) < PasswordMin {
		return ErrInvalidPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("worker: bcrypt: %w", err)
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE workers SET password_hash = $2 WHERE id = $1`, workerID, string(hash))
	if err != nil {
		return fmt.Errorf("worker: set password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// VerifyPassword 校验师傅登录密码(仅读哈希,不暴露 password_hash)。
// 未设置密码(空哈希)或比对不匹配返回 false;仅查库错误返回 err。
func (s *PGStore) VerifyPassword(ctx context.Context, workerID int64, password string) (bool, error) {
	var hash string
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(password_hash, '') FROM workers WHERE id = $1`, workerID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("worker: verify password: %w", err)
	}
	if hash == "" {
		return false, nil
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil, nil
}
