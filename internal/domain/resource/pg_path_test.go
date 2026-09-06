package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 链路反查单测:mock 两层查询(端口行/归属+上级),断言逐跳顺序/类型/编码/状态/占用与断点标记。
const pathPortCols = "port_code,resource_id,status,order_id,order_no,pon_frame,pon_slot,pon_port"
const pathResCols = "id,code,name,type,parent_id,status,p_id,p_code,p_name,p_type,p_status"

func pathPortRow(code, status string, resID, orderID int64, orderNo string, f, s, p int16) *pgxmock.Rows {
	return pgxmock.NewRows([]string{"port_code", "resource_id", "status", "order_id", "order_no", "pon_frame", "pon_slot", "pon_port"}).
		AddRow(code, resID, status, orderID, orderNo, f, s, p)
}

func pathResRows(ownerID int64, ownerCode, ownerType string, parentID int64, parentCode string) *pgxmock.Rows {
	ownerStatus, parentType, parentStatus := "ONLINE", "OLT", "ONLINE"
	parentName := "上级OLT"
	if parentID == 0 {
		parentCode, parentName, parentType, parentStatus = "", "", "", ""
	}
	return pgxmock.NewRows([]string{"id", "code", "name", "type", "parent_id", "status", "p_id", "p_code", "p_name", "p_type", "p_status"}).
		AddRow(ownerID, ownerCode, ownerCode+"名", ownerType, parentID, ownerStatus, parentID, parentCode, parentName, parentType, parentStatus)
}

func expectPortQuery(mock pgxmock.PgxPoolIface, arg any, row *pgxmock.Rows) {
	mock.ExpectQuery("FROM ports p LEFT JOIN orders o").WithArgs(arg).WillReturnRows(row)
}

func expectOwnerQuery(mock pgxmock.PgxPoolIface, resID int64, rows *pgxmock.Rows) {
	mock.ExpectQuery("FROM resources r LEFT JOIN resources up").WithArgs(resID).WillReturnRows(rows)
}

func TestPortPath_FullChain(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	expectPortQuery(mock, "P-SPL01-01", pathPortRow("P-SPL01-01", "USED", 2, 9, "ORD-1", 0, 1, 1))
	expectOwnerQuery(mock, 2, pathResRows(2, "SPL-01", "SPLITTER", 1, "OLT-01"))
	p, err := NewPGStore(mock).PortPath(context.Background(), "P-SPL01-01")
	if err != nil {
		t.Fatalf("PortPath: %v", err)
	}
	wantKinds := []string{HopKindPort, "SPLITTER", HopKindPONPort, HopKindOLT}
	wantCodes := []string{"P-SPL01-01", "SPL-01", "NA-0-1-1", "OLT-01"}
	for i, h := range p.Hops {
		if h.Kind != wantKinds[i] || h.Code != wantCodes[i] || h.Missing {
			t.Fatalf("hop %d = %+v", i, h)
		}
	}
	if !p.Complete {
		t.Fatal("want complete")
	}
	if p.Hops[0].Status != "USED" || p.Hops[0].OccupiedBy == nil || p.Hops[0].OccupiedBy.OrderNo != "ORD-1" {
		t.Fatalf("port hop = %+v", p.Hops[0])
	}
	if p.Hops[3].Status != "ONLINE" {
		t.Fatalf("olt hop = %+v", p.Hops[3])
	}
}

func TestPortPath_SplitterNoParent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	expectPortQuery(mock, "P-SPL02-01", pathPortRow("P-SPL02-01", "IDLE", 3, 0, "", 0, 1, 1))
	expectOwnerQuery(mock, 3, pathResRows(3, "SPL-02", "SPLITTER", 0, ""))
	p, err := NewPGStore(mock).PortPath(context.Background(), "P-SPL02-01")
	if err != nil {
		t.Fatalf("PortPath: %v", err)
	}
	if p.Complete {
		t.Fatal("want incomplete")
	}
	olt := p.Hops[len(p.Hops)-1]
	if olt.Kind != HopKindOLT || !olt.Missing || olt.Reason != BreakSplitterNoParent {
		t.Fatalf("olt hop = %+v", olt)
	}
	if p.Hops[2].Kind != HopKindPONPort || p.Hops[2].Missing || p.Hops[2].Code != "NA-0-1-1" {
		t.Fatalf("pon hop = %+v", p.Hops[2])
	}
}

func TestPortPath_PONUnassigned(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	expectPortQuery(mock, "P-SPL01-02", pathPortRow("P-SPL01-02", "IDLE", 2, 0, "", -1, -1, -1))
	expectOwnerQuery(mock, 2, pathResRows(2, "SPL-01", "SPLITTER", 1, "OLT-01"))
	p, err := NewPGStore(mock).PortPath(context.Background(), "P-SPL01-02")
	if err != nil {
		t.Fatalf("PortPath: %v", err)
	}
	if p.Complete {
		t.Fatal("want incomplete")
	}
	pon := p.Hops[2]
	if pon.Kind != HopKindPONPort || !pon.Missing || pon.Reason != BreakPONUnassigned {
		t.Fatalf("pon hop = %+v", pon)
	}
	if p.Hops[3].Missing || p.Hops[3].Code != "OLT-01" {
		t.Fatalf("olt hop = %+v", p.Hops[3])
	}
}

func TestPortPath_OrphanPort(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	expectPortQuery(mock, int64(7), pathPortRow("P-ORPHAN-01", "IDLE", 99, 0, "", -1, -1, -1))
	expectOwnerQuery(mock, 99, pgxmock.NewRows([]string{"id", "code", "name", "type", "parent_id", "status", "p_id", "p_code", "p_name", "p_type", "p_status"}))
	p, err := NewPGStore(mock).PortPath(context.Background(), "7")
	if err != nil {
		t.Fatalf("PortPath: %v", err)
	}
	if p.Complete {
		t.Fatal("want incomplete")
	}
	if p.Hops[1].Kind != "SPLITTER" || !p.Hops[1].Missing || p.Hops[1].Reason != BreakPortSplitterMissing {
		t.Fatalf("splitter hop = %+v", p.Hops[1])
	}
	if p.Hops[3].Reason != BreakOLTUnreachable {
		t.Fatalf("olt hop = %+v", p.Hops[3])
	}
}

func TestPortPath_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	expectPortQuery(mock, "P-NONE", pgxmock.NewRows([]string{"port_code"}))
	_, err = NewPGStore(mock).PortPath(context.Background(), "P-NONE")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestAssemblePortPath_DirectOLT(t *testing.T) {
	p := &portPathRow{PortCode: "P-OLT-01", ResourceID: 1, Status: "IDLE", PONFrame: -1, PONSlot: -1, PONPort: -1}
	link := &resLinkRow{Owner: &Resource{ID: 1, Code: "OLT-01", Type: HopKindOLT, Status: "ONLINE"}}
	got := assemblePortPath(p, link)
	if len(got.Hops) != 2 || !got.Complete {
		t.Fatalf("got %+v", got)
	}
	if got.Hops[1].Kind != HopKindOLT || got.Hops[1].Code != "OLT-01" {
		t.Fatalf("olt hop = %+v", got.Hops[1])
	}
}
