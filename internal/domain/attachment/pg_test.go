package attachment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// fakeRow 实现 pgx.Row,可编程 Scan 结果。
type fakeRow struct {
	err error // Scan 直接返回该错误
	at  *Attachment
	ts  pgtype.Timestamptz
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if r.at != nil {
		*(dest[0].(*int64)) = r.at.ID
		*(dest[1].(*string)) = r.at.ObjectKey
		*(dest[2].(*string)) = r.at.FileName
		*(dest[3].(*string)) = r.at.ContentType
		*(dest[4].(*int64)) = r.at.SizeBytes
		*(dest[5].(*string)) = r.at.UploaderType
		*(dest[6].(*int64)) = r.at.UploaderID
		*(dest[7].(*pgtype.Timestamptz)) = r.ts
	}
	return nil
}

// fakeRows 实现 pgx.Rows。
type fakeRows struct {
	items   []*fakeRow
	idx     int
	scanErr bool
	nextErr error // 迭代结束后的 Err
	closed  bool
}

func (r *fakeRows) Next() bool { r.idx++; return r.idx <= len(r.items) }

func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr {
		return errors.New("scan boom")
	}
	return r.items[r.idx-1].Scan(dest...)
}

func (r *fakeRows) Err() error                                   { return r.nextErr }
func (r *fakeRows) Close()                                       { r.closed = true }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }

// fakeDB 实现 dbtx。
type fakeDB struct {
	row      *fakeRow
	rows     *fakeRows
	queryErr error
}

func (d *fakeDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if d.queryErr != nil {
		return nil, d.queryErr
	}
	return d.rows, nil
}

func (d *fakeDB) QueryRow(context.Context, string, ...any) pgx.Row { return d.row }

func (d *fakeDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func sampleAt() *Attachment {
	return &Attachment{ID: 7, ObjectKey: "20260101/ab", FileName: "f.pdf",
		ContentType: "application/pdf", SizeBytes: 3, UploaderType: "worker", UploaderID: 9}
}

func TestPGStoreCreate(t *testing.T) {
	db := &fakeDB{row: &fakeRow{at: sampleAt(), ts: ts(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))}}
	got, err := NewPGStore(db).Create(context.Background(), &Attachment{
		ObjectKey: "k", FileName: "f.pdf", ContentType: "application/pdf",
		SizeBytes: 3, UploaderType: "worker", UploaderID: 9})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.ID != 7 || got.CreatedAt != "2026-01-01T00:00:00Z" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestPGStoreCreateInvalidUploader(t *testing.T) {
	if _, err := NewPGStore(&fakeDB{}).Create(context.Background(), &Attachment{UploaderType: "x"}); !errors.Is(err, ErrInvalidUploader) {
		t.Fatalf("want ErrInvalidUploader, got %v", err)
	}
}

func TestPGStoreGet(t *testing.T) {
	db := &fakeDB{row: &fakeRow{at: sampleAt(), ts: ts(time.Now())}}
	if _, err := NewPGStore(db).Get(context.Background(), 7); err != nil {
		t.Fatalf("get: %v", err)
	}

	db = &fakeDB{row: &fakeRow{err: pgx.ErrNoRows}}
	if _, err := NewPGStore(db).Get(context.Background(), 404); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	db = &fakeDB{row: &fakeRow{err: errors.New("boom")}}
	if _, err := NewPGStore(db).Get(context.Background(), 1); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("want scan error, got %v", err)
	}
}

func TestPGStoreListByUploader(t *testing.T) {
	rows := &fakeRows{items: []*fakeRow{{at: sampleAt(), ts: ts(time.Now())}, {at: sampleAt()}}}
	got, err := NewPGStore(&fakeDB{rows: rows}).ListByUploader(context.Background(), "worker", 9, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 || got[0].CreatedAt == "" || got[1].CreatedAt != "" {
		t.Fatalf("unexpected: %+v", got)
	}
	if !rows.closed {
		t.Fatal("rows not closed")
	}

	// limit 钳制边界不 panic 即可(参数化 SQL 无需断言)。
	if _, err := NewPGStore(&fakeDB{rows: &fakeRows{}}).ListByUploader(context.Background(), "worker", 9, 0); err != nil {
		t.Fatalf("clamp: %v", err)
	}
	if _, err := NewPGStore(&fakeDB{rows: &fakeRows{}}).ListByUploader(context.Background(), "worker", 9, 999); err != nil {
		t.Fatalf("clamp: %v", err)
	}

	if _, err := NewPGStore(&fakeDB{queryErr: errors.New("q")}).ListByUploader(context.Background(), "worker", 9, 1); err == nil {
		t.Fatal("query error should fail")
	}
	if _, err := NewPGStore(&fakeDB{rows: &fakeRows{items: []*fakeRow{{}}, scanErr: true}}).ListByUploader(context.Background(), "worker", 9, 1); err == nil {
		t.Fatal("scan error should fail")
	}
	if _, err := NewPGStore(&fakeDB{rows: &fakeRows{nextErr: errors.New("e")}}).ListByUploader(context.Background(), "worker", 9, 1); err == nil {
		t.Fatal("rows.Err should fail")
	}
}

// execDB 记录 Exec 结果以驱动 Delete 断言。
type execDB struct {
	fakeDB
	affected int64
	execErr  error
}

func (d *execDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if d.execErr != nil {
		return pgconn.CommandTag{}, d.execErr
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func TestPGStoreDelete(t *testing.T) {
	db := &execDB{affected: 1}
	if err := NewPGStore(db).Delete(context.Background(), 7); err != nil {
		t.Fatalf("delete: %v", err)
	}
	db = &execDB{execErr: errors.New("boom")}
	if err := NewPGStore(db).Delete(context.Background(), 7); err == nil {
		t.Fatal("exec error should fail")
	}
}

func TestPGStoreGetByIDs(t *testing.T) {
	rows := &fakeRows{items: []*fakeRow{{at: sampleAt(), ts: ts(time.Now())}}}
	got, err := NewPGStore(&fakeDB{rows: rows}).GetByIDs(context.Background(), []int64{7, 7, 0, -1})
	if err != nil {
		t.Fatalf("get-by-ids: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("unexpected: %+v", got)
	}
	empty, err := NewPGStore(&fakeDB{rows: &fakeRows{}}).GetByIDs(context.Background(), nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty ids: %v %+v", err, empty)
	}
	if _, err := NewPGStore(&fakeDB{queryErr: errors.New("q")}).GetByIDs(context.Background(), []int64{7}); err == nil {
		t.Fatal("query error should fail")
	}
}

// countRow 同时充当 count(*) 的 QueryRow 与数据行。
type countRow struct {
	fakeRow
	total int64
}

func (r *countRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*int)) = int(r.total)
	return nil
}

// listDB 让 QueryRow(count) 与 Query(数据) 各走各的桩。
type listDB struct {
	fakeDB
	count *countRow
}

func (d *listDB) QueryRow(context.Context, string, ...any) pgx.Row { return d.count }

func TestPGStoreList(t *testing.T) {
	db := &listDB{count: &countRow{total: 3}, fakeDB: fakeDB{rows: &fakeRows{items: []*fakeRow{{at: sampleAt()}}}}}
	items, total, err := NewPGStore(db).List(context.Background(), ListFilter{
		UploaderType: "worker", UploaderID: 9, Keyword: " f ", Limit: 10, Offset: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 3 || len(items) != 1 {
		t.Fatalf("unexpected: total=%d items=%+v", total, items)
	}
	// limit 钳制与负 offset 不 panic。
	if _, _, err := NewPGStore(db).List(context.Background(), ListFilter{Limit: 0, Offset: -5}); err != nil {
		t.Fatalf("clamp: %v", err)
	}
	// count 失败必须返回错误。
	bad := &listDB{count: &countRow{fakeRow: fakeRow{err: errors.New("c")}}}
	if _, _, err := NewPGStore(bad).List(context.Background(), ListFilter{}); err == nil {
		t.Fatal("count error should fail")
	}
	if _, _, err := NewPGStore(&listDB{count: &countRow{fakeRow: fakeRow{err: errors.New("q")}}}).List(context.Background(), ListFilter{}); err == nil {
		t.Fatal("query error should fail")
	}
}
