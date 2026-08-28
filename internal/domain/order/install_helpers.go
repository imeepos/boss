package order

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrInstallInput 输入校验失败(业务拒绝,非 500)。
// 与 ErrOrderNotFound / ErrOrderState 平级;handler 映射 42200。
var ErrInstallInput = errors.New("order: install invalid input")

// isUniqueViolation 判断 23505 + 指定 constraint_name。
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return pgErr.ConstraintName == constraint
}

// photosToJSON 把 []int64 编码为 JSON 数组字符串("[]" 当空)。
func photosToJSON(ids []int64) (string, error) {
	if ids == nil {
		return "[]", nil
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// jsonToPhotos 反解 photos JSON 数组;失败返回空 slice + nil error(原始字节已落库)。
func jsonToPhotos(b []byte) ([]int64, error) {
	if len(b) == 0 {
		return []int64{}, nil
	}
	out := []int64{}
	if err := json.Unmarshal(b, &out); err != nil {
		return []int64{}, nil
	}
	return out, nil
}

// nullIfZero zero 时间归 NULL(防止 0001-01-01 落库)。
func nullIfZero(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
