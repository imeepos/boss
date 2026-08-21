-- E15:换件流水改 numeric id 强引用——000074 用 ticket_no/epc 字符串软引用,
-- 违反 data-layers 附录 A 修复口径「id 强引用 + code 仅展示冗余」;
-- EPC 当关联码是附录 A「资产码混用」同类复发。补三列 id,ticket_no/epc 留展示快照。
BEGIN;

ALTER TABLE worker_replace_logs
    ADD COLUMN dispatch_ticket_id BIGINT REFERENCES dispatch_tickets(id),
    ADD COLUMN old_tag_id         BIGINT REFERENCES tags(id),
    ADD COLUMN new_tag_id         BIGINT REFERENCES tags(id);

UPDATE worker_replace_logs r SET dispatch_ticket_id = t.id
FROM dispatch_tickets t WHERE r.ticket_no = t.ticket_no;

UPDATE worker_replace_logs r SET old_tag_id = tg.id
FROM tags tg WHERE r.old_epc = tg.epc_code AND r.old_epc <> '';

UPDATE worker_replace_logs r SET new_tag_id = tg.id
FROM tags tg WHERE r.new_epc = tg.epc_code AND r.new_epc <> '';

CREATE INDEX idx_worker_replace_ticket_id ON worker_replace_logs(dispatch_ticket_id);

COMMIT;
