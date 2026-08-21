BEGIN;

ALTER TABLE verifications
    DROP COLUMN reject_reason,
    DROP COLUMN id_card_front_id,
    DROP COLUMN id_card_back_id;

COMMIT;
