package odn

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// facilitySeqSpec/FormatFacilityCode 纯函数回归:模式拼装、序号切片口径、容量上限、非法 kind。
func TestFacilitySeqSpec(t *testing.T) {
	t.Run("P/MH 网格模式:序号从第 4 位起 3 位", func(t *testing.T) {
		spec, err := facilitySeqSpec(KindPole, 12)
		if err != nil || spec.pattern != "^P12[0-9]{3}$" ||
			spec.offset != 4 || spec.digits != 3 || spec.limit != 999 {
			t.Fatalf("spec=%+v err=%v", spec, err)
		}
	})
	t.Run("MH 双字符前缀:序号从第 5 位起", func(t *testing.T) {
		spec, err := facilitySeqSpec(KindManhole, 91)
		if err != nil || spec.pattern != "^MH91[0-9]{3}$" || spec.offset != 5 || spec.digits != 3 {
			t.Fatalf("spec=%+v err=%v", spec, err)
		}
	})
	t.Run("网格越界拒绝", func(t *testing.T) {
		if _, err := facilitySeqSpec(KindManhole, 0); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("err=%v", err)
		}
		if _, err := facilitySeqSpec(KindManhole, 100); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("TW/CLS/TBX 顺序模式:序号紧随前缀", func(t *testing.T) {
		spec, err := facilitySeqSpec(KindTower, 0)
		if err != nil || spec.pattern != "^TW[0-9]{5}$" ||
			spec.offset != 3 || spec.digits != 5 || spec.limit != 99999 {
			t.Fatalf("spec=%+v err=%v", spec, err)
		}
		if spec, err := facilitySeqSpec(KindClosure, 0); err != nil || spec.offset != 4 {
			t.Fatalf("spec=%+v err=%v", spec, err)
		}
	})
	t.Run("非法 kind 拒绝", func(t *testing.T) {
		if _, err := facilitySeqSpec("XX", 1); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("err=%v", err)
		}
	})
}

// facilitySeqSpec 的 offset/digits 必须与 ValidateFacilityCode 解析口径逐字对齐,
// 否则取号出的号会被入库校验拒绝(或反之误报用尽)。
func TestFacilitySeqSpecMatchesValidate(t *testing.T) {
	cases := []struct {
		kind string
		grid int16
		seq  int
	}{
		{KindPole, 91, 13}, {KindPole, 1, 2}, {KindManhole, 99, 999},
		{KindTower, 0, 1}, {KindClosure, 0, 12345}, {KindTermBox, 0, 99999},
	}
	for _, c := range cases {
		code := FormatFacilityCode(c.kind, c.grid, c.seq)
		spec, err := facilitySeqSpec(c.kind, c.grid)
		if err != nil {
			t.Fatalf("%s: spec err=%v", code, err)
		}
		kind, grid, seq, err := ValidateFacilityCode(code)
		if err != nil || kind != c.kind || grid != c.grid || seq != c.seq {
			t.Fatalf("%s: validate kind=%s grid=%d seq=%d err=%v", code, kind, grid, seq, err)
		}
		// SQL SUBSTRING(code FROM offset FOR digits) 的等价 Go 切片同样得出 seq。
		cut := code[spec.offset-1 : spec.offset-1+spec.digits]
		if len(code) != spec.offset-1+spec.digits {
			t.Fatalf("%s: offset/digits 未覆盖整码 spec=%+v", code, spec)
		}
		if got := mustAtoiTest(t, cut); got != c.seq {
			t.Fatalf("%s: 切片序号=%d want %d", code, got, c.seq)
		}
	}
}

func mustAtoiTest(t *testing.T, s string) int {
	t.Helper()
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			t.Fatalf("非数字序号段 %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func TestFormatFacilityCode(t *testing.T) {
	if got := FormatFacilityCode(KindPole, 12, 1); got != "P12001" {
		t.Fatalf("got=%q", got)
	}
	if got := FormatFacilityCode(KindManhole, 3, 47); got != "MH03047" {
		t.Fatalf("got=%q", got)
	}
	if got := FormatFacilityCode(KindClosure, 0, 12); got != "CLS00012" {
		t.Fatalf("got=%q", got)
	}
}

// nextSeqSQL SUBSTRING 取号语句的 pgxmock 匹配式(圆括号在正则里是分组符,须转义)。
const nextSeqSQL = `SELECT COALESCE[(]MAX[(]SUBSTRING[(]code FROM \$2 FOR \$3[)]::int[)],0\)`

// NextFacilityCode 取号口径回归(W7 实锤缺陷):序号=前缀+网格之后的 3 位,
// 旧实现按 '[0-9]+$' 贪婪取数把网格位并入序号,P91011 读成 91011 致网格 91 误报用尽。
func TestNextFacilityCode(t *testing.T) {
	ctx := context.Background()

	t.Run("P91 网格:序号不含网格位,现存 012 顺延 P91013", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		// 断言下发参数:网格段限定正则 + 序号起始位 4 + 位数 3(不是尾部贪婪数字)。
		mock.ExpectQuery(nextSeqSQL).
			WithArgs("^P91[0-9]{3}$", 4, 3).
			WillReturnRows(mock.NewRows([]string{"max"}).AddRow(12))

		got, err := nextFacilityCode(ctx, mock, KindPole, 91)
		if err != nil || got != "P91013" {
			t.Fatalf("got=%q err=%v, want P91013/nil", got, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("P91 网格真满 999:才回 ErrGridFull", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(nextSeqSQL).
			WithArgs("^P91[0-9]{3}$", 4, 3).
			WillReturnRows(mock.NewRows([]string{"max"}).AddRow(999))

		got, err := nextFacilityCode(ctx, mock, KindPole, 91)
		if !errors.Is(err, ErrGridFull) {
			t.Fatalf("got=%q err=%v, want ErrGridFull", got, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("P01 网格正常递增:001 顺延 P01002", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(nextSeqSQL).
			WithArgs("^P01[0-9]{3}$", 4, 3).
			WillReturnRows(mock.NewRows([]string{"max"}).AddRow(1))

		got, err := nextFacilityCode(ctx, mock, KindPole, 1)
		if err != nil || got != "P01002" {
			t.Fatalf("got=%q err=%v, want P01002/nil", got, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("跨网格互不影响:各自 pattern 独立取号", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(nextSeqSQL).
			WithArgs("^P07[0-9]{3}$", 4, 3).
			WillReturnRows(mock.NewRows([]string{"max"}).AddRow(500))
		mock.ExpectQuery(nextSeqSQL).
			WithArgs("^P08[0-9]{3}$", 4, 3).
			WillReturnRows(mock.NewRows([]string{"max"}).AddRow(3))

		g7, err := nextFacilityCode(ctx, mock, KindPole, 7)
		if err != nil || g7 != "P07501" {
			t.Fatalf("g7=%q err=%v, want P07501/nil", g7, err)
		}
		g8, err := nextFacilityCode(ctx, mock, KindPole, 8)
		if err != nil || g8 != "P08004" {
			t.Fatalf("g8=%q err=%v, want P08004/nil", g8, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("网格无既有设施:从 001 起", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(nextSeqSQL).
			WithArgs("^MH42[0-9]{3}$", 5, 3).
			WillReturnRows(mock.NewRows([]string{"max"}).AddRow(0))

		got, err := nextFacilityCode(ctx, mock, KindManhole, 42)
		if err != nil || got != "MH42001" {
			t.Fatalf("got=%q err=%v, want MH42001/nil", got, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("TW 顺序模式:5 位序号紧随前缀", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(nextSeqSQL).
			WithArgs("^TW[0-9]{5}$", 3, 5).
			WillReturnRows(mock.NewRows([]string{"max"}).AddRow(1200))

		got, err := nextFacilityCode(ctx, mock, KindTower, 0)
		if err != nil || got != "TW01201" {
			t.Fatalf("got=%q err=%v, want TW01201/nil", got, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("非法 kind:不发查询直接拒绝", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		if _, err := nextFacilityCode(ctx, mock, "XX", 1); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("err=%v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
