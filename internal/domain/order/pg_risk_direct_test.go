package order

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// riskParamsStub 参数桩:缺 key 视为未配置(走默认)。
type riskParamsStub struct{ values map[string]string }

func (s riskParamsStub) GetParam(_ context.Context, key string) (string, error) {
	v, ok := s.values[key]
	if !ok {
		return "", errors.New("param not found")
	}
	return v, nil
}

func riskCountMock(mock pgxmock.PgxPoolIface, arg int64, counts ...int64) {
	for _, n := range counts {
		mock.ExpectQuery(`SELECT count\(\*\)`).
			WithArgs(arg).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(n))
	}
}

func TestCheckDirectRisk(t *testing.T) {
	req := SubmitReq{CustomerID: 213, OfferID: 101, AddressID: 3988, ChannelID: 104}

	t.Run("默认阈值且双计数未超限-放行", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		riskCountMock(mock, 213, 2)
		riskCountMock(mock, 3988, 1) // phone 2<5, address 1<3
		s := NewPGStore(mock, stubExists{}, riskParamsStub{})
		if err := s.checkDirectRisk(context.Background(), req); err != nil {
			t.Fatalf("want pass, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("同号 24h 多单达阈值-拦截 PHONE_CAP", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		riskCountMock(mock, 213, 5) // phone 命中即短路,不再查 address
		s := NewPGStore(mock, stubExists{}, riskParamsStub{})
		err = s.checkDirectRisk(context.Background(), req)
		if !errors.Is(err, ErrDirectPhoneCap) {
			t.Fatalf("want ErrDirectPhoneCap, got %v", err)
		}
		var pe *CapExceeded
		if !errors.As(err, &pe) || pe.Kind != CapKindPhone || pe.Count != 5 || pe.Cap != 5 {
			t.Fatalf("want CapExceeded phone 5/5, got %v", err)
		}
		if strings.Contains(err.Error(), "order: order:") || !strings.Contains(err.Error(), "count=5") {
			t.Fatalf("Error() 文案异常: %q", err.Error())
		}
	})

	t.Run("同地址在途达阈值-拦截 ADDRESS_CAP", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		riskCountMock(mock, 213, 1)
		riskCountMock(mock, 3988, 3) // phone 1<5, address 3>=3
		s := NewPGStore(mock, stubExists{}, riskParamsStub{})
		err = s.checkDirectRisk(context.Background(), req)
		if !errors.Is(err, ErrDirectAddressCap) {
			t.Fatalf("want ErrDirectAddressCap, got %v", err)
		}
		var ae *CapExceeded
		if !errors.As(err, &ae) || ae.Kind != CapKindAddress || ae.Count != 3 || ae.Cap != 3 {
			t.Fatalf("want CapExceeded address 3/3, got %v", err)
		}
	})

	t.Run("开关关闭-不查库直接放行", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		s := NewPGStore(mock, stubExists{}, riskParamsStub{values: map[string]string{
			"risk.direct.enabled": "false",
		}})
		if err := s.checkDirectRisk(context.Background(), req); err != nil {
			t.Fatalf("want pass, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("disabled 不应发起 count 查询: %v", err)
		}
	})

	t.Run("自定义阈值经 biz_params 生效", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		riskCountMock(mock, 213, 2) // phone cap=2: count 2>=2 命中
		s := NewPGStore(mock, stubExists{}, riskParamsStub{values: map[string]string{
			"risk.direct.phoneCap": "2",
		}})
		err = s.checkDirectRisk(context.Background(), req)
		if !errors.Is(err, ErrDirectPhoneCap) {
			t.Fatalf("want ErrDirectPhoneCap with custom cap, got %v", err)
		}
	})
}
