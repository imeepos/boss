BEGIN;

ALTER TABLE worker_registrations
    DROP CONSTRAINT worker_registrations_group_id_fk;

ALTER TABLE worker_registrations
    ALTER COLUMN group_id  SET NOT NULL,
    ALTER COLUMN region_id SET NOT NULL;

ALTER TABLE worker_registrations
    ADD CONSTRAINT worker_registrations_group_id_fkey
        FOREIGN KEY (group_id) REFERENCES worker_groups(id);

COMMIT;
