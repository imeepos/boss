package order

// 环节7 回归:PreConfigOLT 必须经 ProvisionTemplateFinder 解析模板,
// 禁止再把 product_offers.id 当 provision_templates.id 传入(102 实测错配事故)。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPreConfigOLTResolvesTemplate 回归:任务 TemplateID 来自解析器而非 offer_id。
func TestPreConfigOLTResolvesTemplate(t *testing.T) {
	mock := newAdvanceMock(t)
	defer mock.Close()

	prov := &stubProvCreator{tplID: 501}
	s := NewPGStore(mock, stubExists{ok: true}, &stubProfileCreator{offer: 301}, prov)
	if err := s.PreConfigOLT(context.Background(), 7); err != nil {
		t.Fatalf("PreConfigOLT: %v", err)
	}
	// 解析器收到 (offerID=301, legalEntityID=1)——LO 账号上的套餐 + 订单法人。
	if prov.resolved != [2]int64{301, 1} {
		t.Fatalf("resolved=%v, want [301 1]", prov.resolved)
	}
	// 任务模板 = 解析结果(501),不是 offer_id(301)。
	if prov.created.TemplateID != 501 || prov.created.TaskNo != "PRV-O7" {
		t.Fatalf("created=%+v, want templateID=501 taskNo=PRV-O7", prov.created)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPreConfigOLTRequiresWiring 回归:模板解析口未接线时显性报错,不静默降级。
func TestPreConfigOLTRequiresWiring(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 只注入 CreateTask 桩(未实现 ProvisionTemplateFinder 的独立类型)。
	s := NewPGStore(mock, stubExists{ok: true}, &stubProfileCreator{}, stubTaskOnly{})
	if err := s.PreConfigOLT(context.Background(), 7); err == nil {
		t.Fatal("want error when template finder not wired")
	}
}

// stubTaskOnly 只实现 ProvisionTaskCreator(验证 extras 注入不连带 tpl)。
type stubTaskOnly struct{ ProvisionTaskCreator }

func (stubTaskOnly) CreateTask(context.Context, ProvisionTask) (int64, error) { return 1, nil }

// TestPreConfigOLTResolveErrorPropagates 回归:解析失败(如法人无可用模板)不推进环节。
func TestPreConfigOLTResolveErrorPropagates(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT customer_id, legal_entity_id FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"customer_id", "legal_entity_id"}).AddRow(int64(1), int64(1)))

	prov := &stubProvCreator{err: errors.New("[provision] TEMPLATE MISSING")}
	s := NewPGStore(mock, stubExists{ok: true}, &stubProfileCreator{}, prov)
	if err := s.PreConfigOLT(context.Background(), 7); err == nil {
		t.Fatal("want error when template resolve fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
