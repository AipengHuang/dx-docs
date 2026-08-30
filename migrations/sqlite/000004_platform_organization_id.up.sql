ALTER TABLE tenants ADD COLUMN platform_organization_id TEXT;
CREATE UNIQUE INDEX tenants_platform_organization_id_idx
    ON tenants (platform_organization_id)
    WHERE platform_organization_id IS NOT NULL;
