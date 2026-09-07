package odn

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// 施工开工/竣工流转(F3 门控 + F6 覆盖联动,P-INFRA-1 W4,迁移 000211)。
// 从 pg_construction.go 分出:开工前置校验许可,竣工事务内设施回填与覆盖联动。

// StartProject 开工 PENDING→BUILDING:F3 门控开启时先校验关联许可
// (ROW 已批准且在有效期/PECE 已盖章/NA,过期自动回写 EXPIRED),
// 通过后校验资源范围非空(ErrEmptyScope, P0-A),单内 PLANNED 设施批量推 IN_BUILD。返回设施翻转数。
func (s *PGStore) StartProject(ctx context.Context, id int64) (int64, error) {
	if s.permitGate {
		if _, err := s.CheckProjectPermits(ctx, id); err != nil {
			log.Printf("[odn-permit] START GATE REJECTED proj=%d: %v", id, err)
			return 0, err
		}
	}
	var items int64
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM construction_items WHERE project_id=$1", id).Scan(&items); err != nil {
		log.Printf("[odn-construction] START SCOPE READ FAILED proj=%d: %v", id, err)
		return 0, fmt.Errorf("odn: start scope read: %w", err)
	}
	if err := ValidateStartReady(items); err != nil {
		log.Printf("[odn-construction] START GATE REJECTED proj=%d items=%d: %v", id, items, err)
		return 0, err
	}
	if err := s.casProjectStatus(ctx, id, CPending, CBuilding); err != nil {
		return 0, err
	}
	tag, err := s.db.Exec(ctx, "UPDATE odn_facility f SET lifecycle_status=$2 "+
		"FROM construction_items i WHERE i.facility_code=f.code AND i.project_id=$1 "+
		"AND f.lifecycle_status=$3", id, LCInBuild, LCPlanned)
	if err != nil {
		log.Printf("[odn-construction] START FLIP FAILED proj=%d: %v", id, err)
		return 0, fmt.Errorf("odn: start flip: %w", err)
	}
	return tag.RowsAffected(), nil
}

// AcceptProject 竣工验收 BUILDING→ACCEPTED(F6,P0-C 前置:无未闭环整改):事务内单内设施批量推 IN_SERVICE,
// 并把关联设施地址覆盖 PENDING→SERVED(联动开关 acceptCoverageLink,默认开)。
// 返回设施翻转数与覆盖联动数;联动动作随竣工审计透出。
func (s *PGStore) AcceptProject(ctx context.Context, id int64, acceptedBy int64, note string) (int64, int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-construction] ACCEPT TX BEGIN FAILED proj=%d: %v", id, err)
		return 0, 0, fmt.Errorf("odn: accept begin: %w", err)
	}
	defer tx.Rollback(ctx)
	var openDefects int64
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM construction_defects WHERE project_id=$1 AND status<>'VERIFIED'", id).Scan(&openDefects); err != nil {
		log.Printf("[odn-quality] ACCEPT DEFECT COUNT FAILED proj=%d: %v", id, err)
		return 0, 0, fmt.Errorf("odn: accept defect count: %w", err)
	}
	if openDefects > 0 {
		log.Printf("[odn-quality] ACCEPT GATE REJECTED proj=%d openDefects=%d", id, openDefects)
		return 0, 0, fmt.Errorf("%w: openDefects=%d", ErrOpenDefects, openDefects)
	}
	tag, err := tx.Exec(ctx, "UPDATE construction_projects SET status=$2, "+
		"asbuilt_note=$3, accepted_by=$4, accepted_at=now(), updated_at=now() "+
		"WHERE id=$1 AND status=$5", id, CAccepted, note, acceptedBy, CBuilding)
	if err != nil {
		return 0, 0, fmt.Errorf("odn: accept project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.projectStatus(ctx, id); err != nil {
			return 0, 0, err
		}
		return 0, 0, ErrInvalidProjStatus
	}
	flipped, err := acceptFlipFacilities(ctx, tx, id)
	if err != nil {
		return 0, 0, err
	}
	served := int64(0)
	if s.acceptCoverageLink {
		served, err = linkCoverageOnAccept(ctx, tx, id)
		if err != nil {
			return 0, 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-construction] ACCEPT COMMIT FAILED proj=%d: %v", id, err)
		return 0, 0, fmt.Errorf("odn: accept commit: %w", err)
	}
	if served > 0 {
		log.Printf("[odn-coverage] ACCEPT LINK proj=%d served=%d", id, served)
	}
	return flipped, served, nil
}

// acceptFlipFacilities 单内 PLANNED/IN_BUILD 设施批量推 IN_SERVICE(as-built 回填)。
func acceptFlipFacilities(ctx context.Context, tx pgx.Tx, id int64) (int64, error) {
	tag, err := tx.Exec(ctx, "UPDATE odn_facility f SET lifecycle_status=$2 "+
		"FROM construction_items i WHERE i.facility_code=f.code AND i.project_id=$1 "+
		"AND f.lifecycle_status = ANY($3)", id, LCInService, []string{LCPlanned, LCInBuild})
	if err != nil {
		log.Printf("[odn-construction] ACCEPT FLIP FAILED proj=%d: %v", id, err)
		return 0, fmt.Errorf("odn: accept flip: %w", err)
	}
	return tag.RowsAffected(), nil
}

// linkCoverageOnAccept F6 裁定(000211):项目关联设施地址覆盖 PENDING→SERVED。
func linkCoverageOnAccept(ctx context.Context, tx pgx.Tx, id int64) (int64, error) {
	tag, err := tx.Exec(ctx, "UPDATE address_coverage c SET status=$3, updated_at=now() "+
		"WHERE c.status=$2 AND c.facility_code IN "+
		"(SELECT i.facility_code FROM construction_items i WHERE i.project_id=$1)", id, CovPending, CovServed)
	if err != nil {
		log.Printf("[odn-coverage] ACCEPT LINK FAILED proj=%d: %v", id, err)
		return 0, fmt.Errorf("odn: accept coverage link: %w", err)
	}
	return tag.RowsAffected(), nil
}
