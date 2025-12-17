-- ============================================================================
-- YugabyteDB Monitoring User Creation Script
--
-- This script creates a read-only monitoring user with access to Global Views
-- for OpenTelemetry metrics collection.
-- ============================================================================

\c "gv$";

-- Step 1: Create monitoring user
DO $$
BEGIN
    CREATE USER nr_monitor WITH PASSWORD 'nr_monitor_2024';
END $$;

-- Step 2: Grant database connection privileges
DO $$
BEGIN

    GRANT CONNECT ON DATABASE "gv$" TO nr_monitor;

    GRANT USAGE ON SCHEMA public TO nr_monitor;

    GRANT USAGE ON SCHEMA gv_history TO nr_monitor;

END $$;

-- Step 3: Grant SELECT on tables and views
DO $$
BEGIN

    -- Grant on existing tables
    GRANT SELECT ON ALL TABLES IN SCHEMA public TO nr_monitor;
    GRANT SELECT ON ALL TABLES IN SCHEMA gv_history TO nr_monitor;

    -- Grant on future tables
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO nr_monitor;
    ALTER DEFAULT PRIVILEGES IN SCHEMA gv_history GRANT SELECT ON TABLES TO nr_monitor;

END $$;

-- Step 4: Grant access to node-specific schemas
DO $$
DECLARE
    schema_name TEXT;
    schema_count INTEGER := 0;
BEGIN
    FOR schema_name IN
        SELECT nspname FROM pg_namespace WHERE nspname LIKE 'gv$%'
    LOOP
        EXECUTE format('GRANT USAGE ON SCHEMA %I TO nr_monitor', schema_name);
        EXECUTE format('GRANT SELECT ON ALL TABLES IN SCHEMA %I TO nr_monitor', schema_name);

        schema_count := schema_count + 1;
    END LOOP;
END $$;

-- Step 5: Verify privileges
DO $$
DECLARE
    table_count INTEGER;
    view_count INTEGER;
    schema_count INTEGER;
BEGIN

    -- Count table privileges
    SELECT COUNT(*) INTO table_count
    FROM information_schema.table_privileges
    WHERE grantee = 'nr_monitor'
    AND privilege_type = 'SELECT';

    -- Count views accessible
    SELECT COUNT(*) INTO view_count
    FROM information_schema.views
    WHERE table_schema = 'public' AND table_name LIKE 'gv$%';

    -- Count schemas with access
    SELECT COUNT(DISTINCT table_schema) INTO schema_count
    FROM information_schema.table_privileges
    WHERE grantee = 'nr_monitor';

END $$;

-- Step 6: Display monitoring user summary
DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Monitoring User Setup Complete';
    RAISE NOTICE '========================================';
    RAISE NOTICE '';
    RAISE NOTICE 'User Details:';
    RAISE NOTICE '  • Username: nr_monitor';
    RAISE NOTICE '  • Password: nr_monitor_2024';
    RAISE NOTICE '  • Database: gv$';
    RAISE NOTICE '  • Access Level: Read-only';
    RAISE NOTICE '';
    RAISE NOTICE 'Granted Privileges:';
    RAISE NOTICE '  ✓ CONNECT on database gv$';
    RAISE NOTICE '  ✓ USAGE on schemas: public, gv_history';
    RAISE NOTICE '  ✓ SELECT on all Global Views';
    RAISE NOTICE '  ✓ SELECT on all node-specific schemas';
    RAISE NOTICE '  ✓ SELECT on gv_history tables';
    RAISE NOTICE '';
    RAISE NOTICE 'Connection String:';
    RAISE NOTICE '  postgresql://nr_monitor:nr_monitor_2024@<host>:5433/gv$';
    RAISE NOTICE '';
    RAISE NOTICE 'Security Notes:';
    RAISE NOTICE '  • User has read-only access';
    RAISE NOTICE '  • Cannot create, modify, or delete data';
    RAISE NOTICE '  • Cannot create databases or roles';
    RAISE NOTICE '  • Change the default password in production!';
    RAISE NOTICE '';
    RAISE NOTICE '========================================';
END $$;
