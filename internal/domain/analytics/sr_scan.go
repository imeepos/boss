package analytics

// database/sql 行扫描辅助(StarRocks 实现)。

import (
	"database/sql"
	"fmt"
)

// scanHeat 热力行 → []HeatCell。
func scanHeat(rows *sql.Rows) ([]HeatCell, error) {
	out := make([]HeatCell, 0)
	for rows.Next() {
		var c HeatCell
		if err := rows.Scan(&c.AddressID, &c.Name, &c.Level, &c.PortsTotal, &c.PortsUsed, &c.Utilization); err != nil {
			return nil, fmt.Errorf("analytics: scan heat: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// scanMaint 维护行 → []MaintenanceItem。
func scanMaint(rows *sql.Rows) ([]MaintenanceItem, error) {
	out := make([]MaintenanceItem, 0)
	for rows.Next() {
		var m MaintenanceItem
		if err := rows.Scan(&m.DeviceNo, &m.DeviceType, &m.HealthScore, &m.FaultCount, &m.AgeYears, &m.Reason, &m.Priority); err != nil {
			return nil, fmt.Errorf("analytics: scan maint: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
