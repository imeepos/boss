package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var ErrTemplateNotFound = errors.New("provision: template not found")

func (s *PGStore) UpdateTemplate(ctx context.Context, t Template) error {
	content := t.Content
	if content == nil {
		content = map[string]any{}
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("provision: encode template content: %w", err)
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE provision_templates SET legal_entity_id=$2, code=$3, name=$4, content=$5::jsonb,
			version=version+1, status=$6, updated_at=now()
		WHERE id=$1`, t.ID, t.LegalEntityID, t.Code, t.Name, string(raw), normalizedStatus(t.Status))
	if err != nil {
		return fmt.Errorf("provision: update template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *PGStore) SetTemplateStatus(ctx context.Context, id int64, status string) error {
	tag, err := s.db.Exec(ctx, `UPDATE provision_templates SET status=$2, updated_at=now() WHERE id=$1`, id, normalizedStatus(status))
	if err != nil {
		return fmt.Errorf("provision: set template status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *PGStore) DeleteTemplate(ctx context.Context, id int64) error {
	var used bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM provision_tasks WHERE template_id=$1)`, id).Scan(&used)
	if err != nil {
		return fmt.Errorf("provision: check template use: %w", err)
	}
	if used {
		return fmt.Errorf("provision: template %d is used", id)
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM provision_templates WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("provision: delete template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func normalizedStatus(status string) string {
	if status == "DISABLED" {
		return "DISABLED"
	}
	return "ENABLED"
}

var _ = pgx.ErrNoRows
