DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('menu:collection-tasks','menu:feedback','menu:knowledge',
    'menu:marketing','menu:marketing-recon','menu:message','menu:servers','menu:service-metrics',
    'menu:storageconfig','menu:worker','menu:worker-ops','menu:worker-reg'));
DELETE FROM permissions WHERE code IN ('menu:collection-tasks','menu:feedback','menu:knowledge',
    'menu:marketing','menu:marketing-recon','menu:message','menu:servers','menu:service-metrics',
    'menu:storageconfig','menu:worker','menu:worker-ops','menu:worker-reg');
