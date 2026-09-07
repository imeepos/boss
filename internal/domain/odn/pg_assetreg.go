package odn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var regEntityRefNum = regexp.MustCompile(`^[0-9]+$`)

// registrationCols 凭证查询列(000215)。
const registrationCols = "id, registration_no, entity_kind, COALESCE(facility_code,''), COALESCE(device_id,0), " +
	"asset_id, source_kind, COALESCE(construction_project_id,0), COALESCE(batch_id,0), value_amount, status, " +
	"COALESCE(reverse_reason,''), COALESCE(remark,''), COALESCE(registered_by,0), " +
	"to_char(registered_at,'YYYY-MM-DD HH24:MI:SS'), COALESCE(reversed_by,0), " +
	"COALESCE(to_char(reversed_at,'YYYY-MM-DD HH24:MI:SS'),'')"

func scanRegistration(row pgx.Row, r *AssetRegistration) error {
	return row.Scan(&r.ID, &r.RegistrationNo, &r.EntityKind, &r.FacilityCode, &r.DeviceID,
		&r.AssetID, &r.SourceKind, &r.ConstructionProjectID, &r.BatchID, &r.ValueAmount, &r.Status,
		&r.ReverseReason, &r.Remark, &r.RegisteredBy, &r.RegisteredAt, &r.ReversedBy, &r.ReversedAt)
}

func nextRegistrationNo() string {
	return fmt.Sprintf("ZG-%s-%05d", time.Now().Format("20060102"), time.Now().UnixNano()%100000)
}

// CreateRegistration 资产化登记:校验对象与资产状态,同事务建凭证+资产置 DEPLOYED+落轨迹。
// 资产侧写路径为跨域 SQL(先例 quadlink/pg_scan.go 000185 装机联动),Go 层两域零 import。
func (s *PGStore) CreateRegistration(ctx context.Context, entityKind, facilityCode string, deviceID, assetID int64,
	sourceKind string, projectID int64, value float64, remark string, accountID int64) (*AssetRegistration, error) {
	if err := ValidateRegistrationInput(entityKind, sourceKind, facilityCode, deviceID, projectID, value); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-assetreg] TX BEGIN FAILED asset=%d: %v", assetID, err)
		return nil, fmt.Errorf("odn: reg begin: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockRegistrationTarget(ctx, tx, entityKind, facilityCode, deviceID); err != nil {
		return nil, err
	}
	var assetStatus string
	var batchID *int64
	err = tx.QueryRow(ctx, `SELECT status, batch_id FROM assets WHERE id=$1 FOR UPDATE`, assetID).
		Scan(&assetStatus, &batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("odn: asset %d not found: %w", assetID, ErrNotFound)
	}
	if err != nil {
		log.Printf("[odn-assetreg] ASSET READ FAILED asset=%d: %v", assetID, err)
		return nil, fmt.Errorf("odn: reg asset read: %w", err)
	}
	if assetStatus != "IN_STOCK" && assetStatus != "IN_TRANSIT" {
		return nil, fmt.Errorf("odn: asset %d status=%s: %w", assetID, assetStatus, ErrRegConflict)
	}
	if sourceKind == RegSourceConstruction {
		if err := requireAcceptedProject(ctx, tx, projectID); err != nil {
			return nil, err
		}
	}
	reg, err := insertRegistration(ctx, tx, entityKind, facilityCode, deviceID, assetID, sourceKind,
		projectID, batchID, value, remark, accountID)
	if err != nil {
		return nil, err
	}
	if err := markAssetDeployed(ctx, tx, assetID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-assetreg] COMMIT FAILED reg=%s: %v", reg.RegistrationNo, err)
		return nil, fmt.Errorf("odn: reg commit: %w", err)
	}
	return reg, nil
}

// lockRegistrationTarget 锁定并校验转固对象存在且未退役(40900/40400)。
func lockRegistrationTarget(ctx context.Context, tx pgx.Tx, entityKind, facilityCode string, deviceID int64) error {
	var status string
	var err error
	if entityKind == RegFacility {
		err = tx.QueryRow(ctx, `SELECT status FROM odn_facility WHERE code=$1 FOR UPDATE`, facilityCode).Scan(&status)
	} else {
		err = tx.QueryRow(ctx, `SELECT status FROM odn_device WHERE id=$1 FOR UPDATE`, deviceID).Scan(&status)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("odn: %s not found: %w", entityKind, ErrNotFound)
	}
	if err != nil {
		log.Printf("[odn-assetreg] TARGET READ FAILED kind=%s: %v", entityKind, err)
		return fmt.Errorf("odn: reg target read: %w", err)
	}
	if status == "RETIRED" {
		return fmt.Errorf("odn: %s retired: %w", entityKind, ErrRegConflict)
	}
	return nil
}

// requireAcceptedProject 施工建成来源前置:项目须存在且 ACCEPTED(竣工才转固)。
func requireAcceptedProject(ctx context.Context, tx pgx.Tx, projectID int64) error {
	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM construction_projects WHERE id=$1`, projectID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("odn: project %d not found: %w", projectID, ErrNotFound)
	}
	if err != nil {
		log.Printf("[odn-assetreg] PROJECT READ FAILED proj=%d: %v", projectID, err)
		return fmt.Errorf("odn: reg project read: %w", err)
	}
	if status != CAccepted {
		return fmt.Errorf("odn: project %d status=%s: %w", projectID, status, ErrRegConflict)
	}
	return nil
}

// insertRegistration 落凭证行(部分唯一索引兜底并发:对象/资产已有 ACTIVE 即 23505)。
func insertRegistration(ctx context.Context, tx pgx.Tx, entityKind, facilityCode string, deviceID, assetID int64,
	sourceKind string, projectID int64, batchID *int64, value float64, remark string, accountID int64) (*AssetRegistration, error) {
	var facilityRef *string
	var deviceRef *int64
	if entityKind == RegFacility {
		facilityRef = &facilityCode
	} else {
		deviceRef = &deviceID
	}
	no := nextRegistrationNo()
	r := &AssetRegistration{RegistrationNo: no, EntityKind: entityKind, AssetID: assetID,
		SourceKind: sourceKind, ConstructionProjectID: projectID, ValueAmount: value,
		Status: RegActive, Remark: remark, RegisteredBy: accountID}
	if batchID != nil {
		r.BatchID = *batchID
	}
	err := tx.QueryRow(ctx, `INSERT INTO odn_asset_registrations
		(registration_no, entity_kind, facility_code, device_id, asset_id, source_kind,
		 construction_project_id, batch_id, value_amount, remark, registered_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, to_char(registered_at,'YYYY-MM-DD HH24:MI:SS')`,
		no, entityKind, facilityRef, deviceRef, assetID, sourceKind,
		nullInt64(projectID), batchID, value, remark, accountID).
		Scan(&r.ID, &r.RegisteredAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("odn: registration duplicate entity/asset: %w", ErrRegConflict)
		}
		log.Printf("[odn-assetreg] INSERT FAILED no=%s asset=%d: %v", no, assetID, err)
		return nil, fmt.Errorf("odn: reg insert: %w", err)
	}
	return r, nil
}

func nullInt64(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}

// markAssetDeployed 转固生效:资产置 DEPLOYED 并落轨迹行(同事务,失败回滚凭证)。
func markAssetDeployed(ctx context.Context, tx pgx.Tx, assetID int64) error {
	tag, err := tx.Exec(ctx,
		`UPDATE assets SET status='DEPLOYED', updated_at=now() WHERE id=$1 AND status IN ('IN_STOCK','IN_TRANSIT')`,
		assetID)
	if err != nil {
		log.Printf("[odn-assetreg] ASSET UPDATE FAILED asset=%d: %v", assetID, err)
		return fmt.Errorf("odn: reg asset update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("odn: asset %d not issuable at commit: %w", assetID, ErrRegConflict)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO asset_lifecycles(asset_id, status, changed_at) VALUES ($1,'DEPLOYED',now())`, assetID); err != nil {
		log.Printf("[odn-assetreg] LIFECYCLE WRITE FAILED asset=%d: %v", assetID, err)
		return fmt.Errorf("odn: reg lifecycle: %w", err)
	}
	return nil
}

// ListRegistrations 凭证列表(entityRef=facility code 或 device id;status 空=全部;近单优先)。
func (s *PGStore) ListRegistrations(ctx context.Context, entityKind, entityRef, status string, limit int) ([]AssetRegistration, error) {
	sql := `SELECT ` + registrationCols + ` FROM odn_asset_registrations WHERE 1=1`
	args := []any{}
	if entityKind == RegFacility || entityKind == RegDevice {
		sql += ` AND entity_kind=` + sqlPlaceholder(len(args)+1)
		args = append(args, entityKind)
	}
	if entityRef != "" {
		if entityKind == RegDevice && regEntityRefNum.MatchString(entityRef) {
			id, _ := strconv.ParseInt(entityRef, 10, 64)
			sql += ` AND device_id=` + sqlPlaceholder(len(args)+1)
			args = append(args, id)
		} else {
			sql += ` AND facility_code=` + sqlPlaceholder(len(args)+1)
			args = append(args, entityRef)
		}
	}
	if status == RegActive || status == RegReversed {
		sql += ` AND status=` + sqlPlaceholder(len(args)+1)
		args = append(args, status)
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	sql += ` ORDER BY id DESC LIMIT ` + sqlPlaceholder(len(args)+1)
	args = append(args, limit)
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		log.Printf("[odn-assetreg] LIST FAILED kind=%s ref=%s: %v", entityKind, entityRef, err)
		return nil, fmt.Errorf("odn: list registrations: %w", err)
	}
	defer rows.Close()
	out := []AssetRegistration{}
	for rows.Next() {
		var r AssetRegistration
		if err := scanRegistration(rows, &r); err != nil {
			return nil, fmt.Errorf("odn: scan registration: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRegistration 凭证详情。
func (s *PGStore) GetRegistration(ctx context.Context, id int64) (*AssetRegistration, error) {
	var r AssetRegistration
	err := scanRegistration(s.db.QueryRow(ctx, `SELECT `+registrationCols+
		` FROM odn_asset_registrations WHERE id=$1`, id), &r)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		log.Printf("[odn-assetreg] GET FAILED id=%d: %v", id, err)
		return nil, fmt.Errorf("odn: get registration: %w", err)
	}
	return &r, nil
}

// ReverseRegistration 冲销:ACTIVE→REVERSED(原因必填);资产仍 DEPLOYED 则回 IN_STOCK+轨迹。
func (s *PGStore) ReverseRegistration(ctx context.Context, id, accountID int64, reason string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-assetreg] TX BEGIN FAILED reverse id=%d: %v", id, err)
		return fmt.Errorf("odn: reg reverse begin: %w", err)
	}
	defer tx.Rollback(ctx)
	var assetID int64
	err = tx.QueryRow(ctx, `UPDATE odn_asset_registrations SET status=$1, reverse_reason=$2,
		reversed_by=$3, reversed_at=now(), updated_at=now()
		WHERE id=$4 AND status=$5 RETURNING asset_id`,
		RegReversed, reason, accountID, id, RegActive).Scan(&assetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("odn: registration %d not active: %w", id, ErrRegState)
	}
	if err != nil {
		log.Printf("[odn-assetreg] REVERSE FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: reg reverse: %w", err)
	}
	tag, err := tx.Exec(ctx,
		`UPDATE assets SET status='IN_STOCK', updated_at=now() WHERE id=$1 AND status='DEPLOYED'`, assetID)
	if err != nil {
		log.Printf("[odn-assetreg] REVERSE ASSET FAILED id=%d asset=%d: %v", id, assetID, err)
		return fmt.Errorf("odn: reg reverse asset: %w", err)
	}
	if tag.RowsAffected() > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO asset_lifecycles(asset_id, status, changed_at) VALUES ($1,'IN_STOCK',now())`, assetID); err != nil {
			log.Printf("[odn-assetreg] REVERSE LIFECYCLE FAILED asset=%d: %v", assetID, err)
			return fmt.Errorf("odn: reg reverse lifecycle: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-assetreg] REVERSE COMMIT FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: reg reverse commit: %w", err)
	}
	return nil
}

// sqlPlaceholder 动态占位符($n),ListRegistrations 组合过滤用。
func sqlPlaceholder(n int) string {
	return "$" + strconv.Itoa(n)
}
