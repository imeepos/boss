package resource

import (
	"context"
	"fmt"
)

// CheckDetail 目标地址的端口库存快照,供管理员核查前预览。
type CheckDetail struct {
	AddressID int64             `json:"addressId"`
	Total     int               `json:"total"`
	Idle      int               `json:"idle"`
	Reserved  int               `json:"reserved"`
	Used      int               `json:"used"`
	Disabled  int               `json:"disabled"`
	IdleCodes []string          `json:"idleCodes"`
	Devices   []CheckDeviceRow  `json:"devices"`
}

// CheckDeviceRow 设备下端口状态汇总。
type CheckDeviceRow struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
	Status string `json:"status"`
	Total int `json:"total"`
	Idle int `json:"idle"`
}

// CheckDetail 查询目标地址下设备及端口状态,不改变订单或资源状态。
func (s *PGStore) CheckDetail(ctx context.Context, addressID int64) (*CheckDetail, error) {
	d := &CheckDetail{AddressID: addressID, IdleCodes: []string{}, Devices: []CheckDeviceRow{}}
	rows, err := s.db.Query(ctx, `
		SELECT r.code, r.name, r.type, r.status,
		       COUNT(p.id), COUNT(p.id) FILTER (WHERE p.status = 'IDLE'),
		       COUNT(p.id) FILTER (WHERE p.status = 'RESERVED'),
		       COUNT(p.id) FILTER (WHERE p.status = 'USED'),
		       COUNT(p.id) FILTER (WHERE p.status = 'DISABLED')
		FROM resources r LEFT JOIN ports p ON p.resource_id = r.id
		WHERE r.address_id = $1 GROUP BY r.id ORDER BY r.id`, addressID)
	if err != nil { return nil, fmt.Errorf("resource: check detail query: %w", err) }
	defer rows.Close()
	for rows.Next() {
		var x CheckDeviceRow
		var reserved, used, disabled int
		if err := rows.Scan(&x.Code, &x.Name, &x.Type, &x.Status, &x.Total, &x.Idle, &reserved, &used, &disabled); err != nil {
			return nil, fmt.Errorf("resource: check detail scan: %w", err)
		}
		d.Total += x.Total
		d.Idle += x.Idle
		d.Reserved += reserved
		d.Used += used
		d.Disabled += disabled
		d.Devices = append(d.Devices, x)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("resource: check detail rows: %w", err) }
	codes, err := s.db.Query(ctx, `SELECT port_code FROM ports WHERE address_id = $1 AND status = 'IDLE' ORDER BY id`, addressID)
	if err != nil { return nil, fmt.Errorf("resource: check detail idle ports: %w", err) }
	defer codes.Close()
	for codes.Next() { var code string; if err := codes.Scan(&code); err != nil { return nil, fmt.Errorf("resource: check detail idle scan: %w", err) }; d.IdleCodes = append(d.IdleCodes, code) }
	return d, codes.Err()
}
