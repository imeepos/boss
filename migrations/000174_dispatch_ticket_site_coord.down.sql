ALTER TABLE dispatch_tickets
    DROP COLUMN IF EXISTS site_lat,
    DROP COLUMN IF EXISTS site_lng;
