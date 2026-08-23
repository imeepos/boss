BEGIN;
DELETE FROM biz_params WHERE key='commission.partnerDefaultRate';
COMMIT;
