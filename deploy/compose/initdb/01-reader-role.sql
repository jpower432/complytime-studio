-- SPDX-License-Identifier: Apache-2.0
-- Create read-only Postgres role for external dashboard clients.

DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'core_reader') THEN
    CREATE ROLE core_reader LOGIN PASSWORD 'complytime-reader-dev';
  END IF;
END $$;

GRANT USAGE ON SCHEMA public TO core_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO core_reader;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO core_reader;
