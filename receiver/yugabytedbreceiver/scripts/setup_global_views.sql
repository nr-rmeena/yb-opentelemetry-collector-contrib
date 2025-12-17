-- ============================================================================
-- YugabyteDB Global View Setup SQL
--
-- This script creates Foreign Data Wrapper (FDW) infrastructure to aggregate
-- metrics from all nodes in a YugabyteDB cluster using Global Views.
-- ============================================================================

-- Step 1: Create dedicated database for Global Views

CREATE DATABASE "gv$";
\c "gv$";

-- Step 2: Enable Foreign Data Wrapper extension

CREATE EXTENSION IF NOT EXISTS postgres_fdw;


-- Step 3: Create foreign servers for each cluster node
DO $$
DECLARE
    host TEXT;
    port TEXT;
    current_db TEXT := current_database();
    server_count INTEGER := 0;
BEGIN

    FOR host, port IN SELECT s.host, s.port FROM yb_servers() s LOOP
        EXECUTE format('
            CREATE SERVER IF NOT EXISTS "gv$%1$s"
            FOREIGN DATA WRAPPER postgres_fdw
            OPTIONS (host %2$L, port %3$L, dbname %4$L)
        ', host, host, port, current_db);

        server_count := server_count + 1;
    END LOOP;
END $$;

-- Step 4: Create user mappings for foreign servers
DO $$
DECLARE
    host TEXT;
    mapping_count INTEGER := 0;
BEGIN

    FOR host IN SELECT s.host FROM yb_servers() s LOOP
        EXECUTE format('
            CREATE USER MAPPING IF NOT EXISTS FOR CURRENT_USER
            SERVER "gv$%1$s"
            OPTIONS (user %2$L, password %3$L)
        ', host, '${dbUser}', '${dbPassword}');

        mapping_count := mapping_count + 1;
    END LOOP;
END $$;

-- Step 5: Create schemas for foreign tables
DO $$
DECLARE
    host TEXT;
    schema_count INTEGER := 0;
BEGIN

    FOR host IN SELECT s.host FROM yb_servers() s LOOP
        EXECUTE format('DROP SCHEMA IF EXISTS "gv$%1$s" CASCADE', host);
        EXECUTE format('CREATE SCHEMA IF NOT EXISTS "gv$%1$s"', host);

        schema_count := schema_count + 1;
    END LOOP;

END $$;

-- Step 6: Import foreign schemas from each node
DO $$
DECLARE
    host TEXT;
    import_count INTEGER := 0;
BEGIN


    FOR host IN SELECT s.host FROM yb_servers() s LOOP
        EXECUTE format('
            IMPORT FOREIGN SCHEMA "pg_catalog"
            LIMIT TO ("pg_stat_activity", "pg_stat_statements", "pg_stat_database")
            FROM SERVER "gv$%1$s" INTO "gv$%1$s"
        ', host);

        import_count := import_count + 1;
    END LOOP;

END $$;


-- Step 7: Create gv_history schema for historical data
DO $$
BEGIN
CREATE SCHEMA IF NOT EXISTS gv_history;
SET search_path TO gv_history;
END $$;

-- Step 8: Create Global Views aggregating data from all nodes
DO $$
DECLARE
    foreign_table_name TEXT;
    view_query TEXT;
    view_count INTEGER := 0;
BEGIN

    FOR foreign_table_name IN
        SELECT DISTINCT t.foreign_table_name
        FROM information_schema.foreign_tables t, yb_servers() s
        WHERE t.foreign_table_schema = format('gv$%1$s', LEFT(s.host, 60))
    LOOP
        EXECUTE format('DROP VIEW IF EXISTS "gv$%1$s"', foreign_table_name);

        SELECT string_agg(
            format('
                SELECT %2$L AS gv$host, %3$L AS gv$zone, %4$L AS gv$region, %5$L AS gv$cloud,
                * FROM "gv$%2$s".%1$I
            ', foreign_table_name, s.host, s.zone, s.region, s.cloud),
            ' UNION ALL '
        )
        INTO view_query
        FROM yb_servers() s;

        EXECUTE format('CREATE OR REPLACE VIEW "gv$%1$s" AS %2$s', foreign_table_name, view_query);

        view_count := view_count + 1;
    END LOOP;

END $$;


-- Step 9: Create historical table for pg_stat_statements
DO $$
BEGIN


    DROP TABLE IF EXISTS gv_history.global_pg_stat_statements;

    CREATE TABLE gv_history.global_pg_stat_statements (
        LIKE pg_stat_statements INCLUDING ALL,
        snapshot_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

    CREATE INDEX idx_snapshot_time ON gv_history.global_pg_stat_statements (snapshot_time);

END $$;

-- Step 10: Display setup summary
DO $$
DECLARE
    server_count INTEGER;
    view_count INTEGER;
    schema_count INTEGER;
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Global Views Setup Complete';
    RAISE NOTICE '========================================';

    -- Count foreign servers
    SELECT COUNT(*) INTO server_count
    FROM pg_foreign_server
    WHERE srvname LIKE 'gv$%';

    -- Count global views
    SELECT COUNT(*) INTO view_count
    FROM information_schema.views
    WHERE table_schema = 'public' AND table_name LIKE 'gv$%';

    -- Count schemas
    SELECT COUNT(*) INTO schema_count
    FROM pg_namespace
    WHERE nspname LIKE 'gv$%';

    RAISE NOTICE '';
    RAISE NOTICE 'Summary:';
    RAISE NOTICE '  • Foreign servers: %', server_count;
    RAISE NOTICE '  • Foreign schemas: %', schema_count;
    RAISE NOTICE '  • Global Views: %', view_count;
    RAISE NOTICE '  • History schema: gv_history';
    RAISE NOTICE '  • History tables: 1';
    RAISE NOTICE '';
    RAISE NOTICE 'Available Global Views:';

    FOR view_name IN
        SELECT table_name
        FROM information_schema.views
        WHERE table_schema = 'public' AND table_name LIKE 'gv$%'
        ORDER BY table_name
    LOOP
        RAISE NOTICE '  ✓ %', view_name;
    END LOOP;

    RAISE NOTICE '';
    RAISE NOTICE '========================================';
END $$;
