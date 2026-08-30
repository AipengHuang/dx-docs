DROP INDEX IF EXISTS tenants_platform_organization_id_idx;
ALTER TABLE tenants DROP COLUMN IF EXISTS platform_organization_id;
