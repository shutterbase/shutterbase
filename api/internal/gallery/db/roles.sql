-- Database roles for the public gallery. Applied once by the DBA / GitOps
-- AFTER the shutterbase release that creates the `galleries` table is live
-- (the grants name the tables). Never applied by the application.
--
--   gallery_owner    owns schema `gallery`; used only by `gallery migrate`
--   gallery_runtime  NOLOGIN group inherited by the Vault-issued web/worker
--                    credentials (database secrets engine: CREATE ROLE ...
--                    IN ROLE gallery_runtime)
--
-- Explicit table grants, no default privileges on `public`: new shutterbase
-- tables (audit logs, persons, api keys, ...) stay invisible to the gallery.

CREATE ROLE gallery_owner NOLOGIN;
CREATE ROLE gallery_runtime NOLOGIN;

CREATE SCHEMA IF NOT EXISTS gallery AUTHORIZATION gallery_owner;
GRANT USAGE ON SCHEMA gallery TO gallery_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE gallery_owner IN SCHEMA gallery
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO gallery_runtime;

GRANT USAGE ON SCHEMA public TO gallery_owner, gallery_runtime;
GRANT SELECT ON galleries, projects, images, image_tags, image_tag_assignments, uploads, cameras
  TO gallery_runtime;
-- Column-level: the gallery credits photographers, nothing more. Every
-- gallery query that touches users selects exactly these columns.
GRANT SELECT (id, first_name, last_name, copyright_tag) ON users TO gallery_runtime;

-- Example Vault database secrets engine role (creation statement):
--   CREATE ROLE "{{name}}" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}' IN ROLE gallery_runtime;
-- Static owner credential for the migration Job:
--   CREATE ROLE gallery_migrator LOGIN PASSWORD '...' IN ROLE gallery_owner; SET ROLE gallery_owner;
