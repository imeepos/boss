BEGIN;
DELETE FROM biz_params WHERE key IN ('fraud.partnerDailyOrderCap', 'fraud.partnerCustomerCooldownHours');
COMMIT;
