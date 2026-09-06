package odn

import (
	"context"
	"fmt"
)

// ImpactByFacility 设施影响面:直接挂接的覆盖地址 + 这些地址上的客户。
// 只读聚合,不写任何表;设施不存在返回 ErrNotFound。
func (s *PGStore) ImpactByFacility(ctx context.Context, facilityCode string) (*ImpactReport, error) {
	fac, err := s.GetFacility(ctx, facilityCode)
	if err != nil {
		return nil, err
	}
	rep := &ImpactReport{FacilityCode: facilityCode, FacilityName: fac.Name, Coverages: []ImpactCoverage{}, Customers: []ImpactCustomer{}}
	rows, err := s.db.Query(ctx, `SELECT c.address_id, COALESCE(a.name,''), c.status
		FROM address_coverage c LEFT JOIN addresses a ON a.id = c.address_id
		WHERE c.facility_code=$1 ORDER BY c.address_id`, facilityCode)
	if err != nil {
		return nil, fmt.Errorf("odn: impact coverages: %w", err)
	}
	addrIDs := []int64{}
	for rows.Next() {
		var ic ImpactCoverage
		if err := rows.Scan(&ic.AddressID, &ic.AddressName, &ic.Status); err != nil {
			rows.Close()
			return nil, fmt.Errorf("odn: impact scan coverage: %w", err)
		}
		addrIDs = append(addrIDs, ic.AddressID)
		rep.Coverages = append(rep.Coverages, ic)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("odn: impact coverages rows: %w", err)
	}
	rows.Close()
	if len(addrIDs) == 0 {
		return rep, nil
	}
	crows, err := s.db.Query(ctx, `SELECT id, COALESCE(name,''), COALESCE(phone,'') FROM customers
		WHERE address_id = ANY($1) ORDER BY id`, addrIDs)
	if err != nil {
		return nil, fmt.Errorf("odn: impact customers: %w", err)
	}
	defer crows.Close()
	for crows.Next() {
		var ic ImpactCustomer
		if err := crows.Scan(&ic.CustomerID, &ic.Name, &ic.Phone); err != nil {
			return nil, fmt.Errorf("odn: impact scan customer: %w", err)
		}
		rep.Customers = append(rep.Customers, ic)
	}
	return rep, crows.Err()
}
