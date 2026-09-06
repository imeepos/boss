package resource

// PON 端到端链路反查(P5-W2,对标 NRM C4 拓扑路径追踪)。
// 沿 ports.resource_id 与 resources.parent_id 派生 端口→分光器→PON口→OLT 逐跳物理链路;
// 只读派生视图,无新表新列;断链只落 missing 跳 + 断点原因码,禁止猜链补链。
// 契约:fields.md §4.2 链路反查视图;路由 GET /ports/:portId/path(portRef 支持端口 ID 或端口编码)。

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// 链路跳类型(派生视图枚举,非状态枚举;资源跳直接取 resources.type 实际值)。
const (
	HopKindPort     = "PORT"
	HopKindSplitter = "SPLITTER"
	HopKindPONPort  = "PON_PORT"
	HopKindOLT      = "OLT"
)

// 断点原因码:只描述库里缺这条边的事实,不做任何推断补链。
const (
	BreakPortSplitterMissing = "PORT_SPLITTER_MISSING" // 孤儿端口:resource_id 无对应资源行
	BreakSplitterNoParent    = "SPLITTER_NO_PARENT"    // 分光器无上级(无 OLT 归属)
	BreakPONUnassigned       = "PON_PORT_UNASSIGNED"   // 端口未配 PON 框/槽/口
	BreakOLTUnreachable      = "OLT_UNREACHABLE"       // 归属资源缺失,无法上溯 OLT
)

// PathOccupant 跳上的占用引用(当前仅端口→订单)。
type PathOccupant struct {
	Kind    string `json:"kind"` // order
	ID      int64  `json:"id"`
	OrderNo string `json:"orderNo,omitempty"`
}

// PathHop 逐跳物理链路单跳;missing=true 时 code/status 为空、reason 非空(断点标记)。
type PathHop struct {
	Seq        int           `json:"seq"`
	Kind       string        `json:"kind"`
	Code       string        `json:"code"`
	Name       string        `json:"name,omitempty"`
	Status     string        `json:"status"` // 端口/设备状态;PON_PORT 无独立状态恒空
	OccupiedBy *PathOccupant `json:"occupiedBy,omitempty"`
	Missing    bool          `json:"missing"`
	Reason     string        `json:"reason,omitempty"` // 断点原因码,仅 missing=true 时非空
}

// PortPath 链路反查结果;complete=无断点且末跳为 OLT。
type PortPath struct {
	PortCode string    `json:"portCode"`
	Complete bool      `json:"complete"`
	Hops     []PathHop `json:"hops"`
}

const portPathSQL = `SELECT p.port_code, p.resource_id, p.status, COALESCE(p.order_id, 0),` +
	`COALESCE(o.order_no, ''), COALESCE(p.pon_frame, -1), COALESCE(p.pon_slot, -1), COALESCE(p.pon_port, -1)` +
	` FROM ports p LEFT JOIN orders o ON o.id = p.order_id WHERE `

const ownerParentSQL = `SELECT r.id, r.code, r.name, r.type, COALESCE(r.parent_id, 0), r.status,` +
	`COALESCE(up.id, 0), COALESCE(up.code, ''), COALESCE(up.name, ''), COALESCE(up.type, ''), COALESCE(up.status, '')` +
	` FROM resources r LEFT JOIN resources up ON up.id = r.parent_id WHERE r.id = $1`

// portPathRow 端口行链路取数投影;PON 三列 -1=未分配(NULL 哨兵)。
type portPathRow struct {
	PortCode   string
	ResourceID int64
	Status     string
	OrderID    int64
	OrderNo    string
	PONFrame   int16
	PONSlot    int16
	PONPort    int16
}

// resLinkRow 归属资源及其上级(一次 JOIN 取两层)。
type resLinkRow struct {
	Owner  *Resource // nil=孤儿端口(归属资源行缺失)
	Parent *Resource // nil=分光器无上级
}

// PortPath 反查端口逐跳物理链路;portRef 支持端口 ID(纯数字)或端口编码。
// 端口不存在返回 ErrNotFound;链路缺失跳以 missing 标记返回,不报错不补链。
func (s *PGStore) PortPath(ctx context.Context, portRef string) (*PortPath, error) {
	p, err := s.getPortPathRow(ctx, portRef)
	if err != nil {
		return nil, err
	}
	link, err := s.getOwnerAndParent(ctx, p.ResourceID)
	if err != nil {
		return nil, err
	}
	return assemblePortPath(p, link), nil
}

// getPortPathRow 按编码或 ID 取端口行并联订单号(占用引用)。
func (s *PGStore) getPortPathRow(ctx context.Context, portRef string) (*portPathRow, error) {
	where, arg := portWhere(portRef)
	var p portPathRow
	err := s.db.QueryRow(ctx, portPathSQL+where, arg).Scan(
		&p.PortCode, &p.ResourceID, &p.Status, &p.OrderID, &p.OrderNo,
		&p.PONFrame, &p.PONSlot, &p.PONPort)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resource: port path %s: %w", portRef, err)
	}
	return &p, nil
}

// portWhere 纯数字按主键,其余按端口编码(端口编码非纯数字,两入口无歧义)。
func portWhere(ref string) (string, any) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil && id > 0 {
		return "p.id = $1", id
	}
	return "p.port_code = $1", ref
}

// getOwnerAndParent 归属资源+上级单查询;归属行缺失返回零行 link(孤儿端口语义)。
func (s *PGStore) getOwnerAndParent(ctx context.Context, resourceID int64) (*resLinkRow, error) {
	var owner, up Resource
	err := s.db.QueryRow(ctx, ownerParentSQL, resourceID).Scan(
		&owner.ID, &owner.Code, &owner.Name, &owner.Type, &owner.ParentID, &owner.Status,
		&up.ID, &up.Code, &up.Name, &up.Type, &up.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return &resLinkRow{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resource: port path owner %d: %w", resourceID, err)
	}
	link := &resLinkRow{}
	if owner.ID > 0 {
		link.Owner = &owner
	}
	if up.ID > 0 {
		link.Parent = &up
	}
	return link, nil
}

// assemblePortPath 组装逐跳视图(纯函数):PORT → [SPLITTER] → PON_PORT → OLT。
// 端口直挂 OLT(无分光器)时链路两跳终止;缺失跳落 missing 标记,禁止补链。
func assemblePortPath(p *portPathRow, link *resLinkRow) *PortPath {
	hops := []PathHop{portHop(p, 1)}
	switch {
	case link.Owner == nil:
		hops = append(hops,
			missingHop(2, HopKindSplitter, BreakPortSplitterMissing),
			ponHop(p, 3),
			missingHop(4, HopKindOLT, BreakOLTUnreachable))
	case link.Owner.Type == HopKindOLT:
		hops = append(hops, resourceHop(link.Owner, 2))
	default:
		hops = append(hops, resourceHop(link.Owner, 2), ponHop(p, 3))
		if link.Parent == nil {
			hops = append(hops, missingHop(4, HopKindOLT, BreakSplitterNoParent))
		} else {
			hops = append(hops, resourceHop(link.Parent, 4))
		}
	}
	return &PortPath{PortCode: p.PortCode, Complete: pathComplete(hops), Hops: hops}
}

// pathComplete 无断点且末跳为 OLT。
func pathComplete(hops []PathHop) bool {
	for i := range hops {
		if hops[i].Missing {
			return false
		}
	}
	return hops[len(hops)-1].Kind == HopKindOLT
}

func portHop(p *portPathRow, seq int) PathHop {
	h := PathHop{Seq: seq, Kind: HopKindPort, Code: p.PortCode, Status: p.Status}
	if p.OrderID > 0 {
		h.OccupiedBy = &PathOccupant{Kind: "order", ID: p.OrderID, OrderNo: p.OrderNo}
	}
	return h
}

func resourceHop(r *Resource, seq int) PathHop {
	return PathHop{Seq: seq, Kind: r.Type, Code: r.Code, Name: r.Name, Status: r.Status}
}

// ponHop PON 口跳:PONID 组装口径同 fields.md §4.4(NA-框-槽-口);未配落断点。
func ponHop(p *portPathRow, seq int) PathHop {
	if p.PONFrame < 0 || p.PONSlot < 0 || p.PONPort < 0 {
		return missingHop(seq, HopKindPONPort, BreakPONUnassigned)
	}
	return PathHop{Seq: seq, Kind: HopKindPONPort,
		Code: fmt.Sprintf("NA-%d-%d-%d", p.PONFrame, p.PONSlot, p.PONPort)}
}

func missingHop(seq int, kind, reason string) PathHop {
	return PathHop{Seq: seq, Kind: kind, Missing: true, Reason: reason}
}
