package resource

// 扩容单写侧补齐:执行=PENDING→DONE(目标设备按 expectedPorts 批量建端口,单请求内完成),
// 驳回=PENDING→DONE。端口码 P-<设备码去横杠>-<序号> 续号(fields.md §4.2 编码规范),
// quad_code 初始=端口码(四码关联建立时细化),IDLE 入池;预占仍走订单状态机,不经此处。

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ExpansionResult 扩容执行结果。
type ExpansionResult struct {
	Created    int      `json:"created"`    // 本次新建端口数
	Existing   int      `json:"existing"`   // 执行前设备已有端口数
	TotalPorts int      `json:"totalPorts"` // 执行后设备端口总数
	PortCodes  []string `json:"portCodes"`  // 本次新建端口码
}

// RejectExpansion 扩容驳回:PENDING→DONE(契约状态仅 PENDING/DOING/DONE,驳回即终态)。
func (s *PGStore) RejectExpansion(ctx context.Context, expansionNo string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE expansions SET status = 'DONE' WHERE expansion_no = $1 AND status = 'PENDING'`, expansionNo)
	if err != nil {
		return fmt.Errorf("resource: reject expansion: %w", err)
	}
	return reviewResult(tag)
}

// ExecuteExpansion 执行扩容单:仅 PENDING 可执行;设备必须存在且归属扩容单同一法人。
// 建满 expectedPorts 才置 DONE;建口失败保留 PENDING(已建端口保留,重试按续号补建)。
func (s *PGStore) ExecuteExpansion(ctx context.Context, expansionNo string, resourceID int64) (*ExpansionResult, error) {
	var legalEntityID, regionID int64
	var expected int32
	var status string
	err := s.db.QueryRow(ctx,
		`SELECT legal_entity_id, region_id, expected_ports, status FROM expansions WHERE expansion_no = $1 FOR UPDATE`,
		expansionNo).Scan(&legalEntityID, &regionID, &expected, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resource: load expansion: %w", err)
	}
	if status != "PENDING" {
		return nil, ErrIllegalTransition
	}
	dev, err := s.GetResource(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("resource: expansion device: %w", err)
	}
	if dev.LegalEntityID != legalEntityID {
		return nil, fmt.Errorf("resource: expansion device %d entity %d != expansion entity %d: %w",
			dev.ID, dev.LegalEntityID, legalEntityID, ErrForeignKeyViolation)
	}
	entityName, err := s.lookupName(ctx, "legal_entities", legalEntityID)
	if err != nil {
		return nil, err
	}
	regionName, err := s.lookupName(ctx, "regions", regionID)
	if err != nil {
		return nil, err
	}
	existing, err := s.ListPorts(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	res := &ExpansionResult{Existing: len(existing), PortCodes: []string{}}
	if need := int(expected) - len(existing); need > 0 {
		codes, err := s.buildExpansionPorts(ctx, dev, existing, legalEntityID, entityName, regionID, regionName, need)
		if err != nil {
			return nil, err
		}
		res.PortCodes = codes
		res.Created = len(codes)
	}
	res.TotalPorts = res.Existing + res.Created
	tag, err := s.db.Exec(ctx,
		`UPDATE expansions SET status = 'DONE' WHERE expansion_no = $1 AND status = 'PENDING'`, expansionNo)
	if err != nil {
		return nil, fmt.Errorf("resource: expansion done: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrIllegalTransition
	}
	return res, nil
}

// buildExpansionPorts 批量建口:续号跳过已占码,单口失败即返回(已建保留,可重试)。
func (s *PGStore) buildExpansionPorts(ctx context.Context, dev *Resource, existing []Port, legalEntityID int64,
	entityName string, regionID int64, regionName string, need int) ([]string, error) {
	seq := nextPortSeq(existing, dev.Code)
	taken := portCodeSet(existing)
	codes := make([]string, 0, need)
	for i := 0; i < need; i++ {
		code := portCodeFor(dev.Code, seq)
		for taken[code] {
			seq++
			code = portCodeFor(dev.Code, seq)
		}
		p := expansionPort(code, dev, legalEntityID, entityName, regionID, regionName)
		if _, err := s.CreatePort(ctx, p); err != nil {
			return nil, fmt.Errorf("resource: expansion build port %s: %w", code, err)
		}
		taken[code] = true
		codes = append(codes, code)
		seq++
	}
	return codes, nil
}

// lookupName 查快照名(法人/区域);缺失返回 ErrNotFound(区域无外键,应用层兜底)。
func (s *PGStore) lookupName(ctx context.Context, table string, id int64) (string, error) {
	var name string
	err := s.db.QueryRow(ctx, `SELECT name FROM `+table+` WHERE id = $1`, id).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("resource: %s %d: %w", table, id, ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("resource: lookup %s %d: %w", table, id, err)
	}
	return name, nil
}

// nextPortSeq 端口续号起点:P-<设备码去横杠>-<N> 形态最大序号+1,无匹配从 1 起。
func nextPortSeq(existing []Port, deviceCode string) int {
	prefix := "P-" + strings.ReplaceAll(deviceCode, "-", "") + "-"
	next := 1
	for _, p := range existing {
		suffix, ok := strings.CutPrefix(p.PortCode, prefix)
		if !ok {
			continue
		}
		if n, err := strconv.Atoi(suffix); err == nil && n >= next {
			next = n + 1
		}
	}
	return next
}

func portCodeSet(existing []Port) map[string]bool {
	set := make(map[string]bool, len(existing))
	for _, p := range existing {
		set[p.PortCode] = true
	}
	return set
}

// portCodeFor 端口编码:P-<设备码去横杠>-<序号>(两位零填充,超 99 自然扩展)。
func portCodeFor(deviceCode string, seq int) string {
	return fmt.Sprintf("P-%s-%02d", strings.ReplaceAll(deviceCode, "-", ""), seq)
}

// expansionPort 执行扩容建的端口:IDLE 入池,quad_code 初始=端口码,区域取扩容单快照。
func expansionPort(code string, dev *Resource, legalEntityID int64, entityName string, regionID int64, regionName string) Port {
	return Port{
		PortCode: code, QuadCode: code,
		ResourceID: dev.ID, LegalEntityID: legalEntityID, LegalEntityName: entityName,
		AddressID: dev.AddressID, RegionID: regionID, RegionName: regionName,
		Status: "IDLE",
	}
}
