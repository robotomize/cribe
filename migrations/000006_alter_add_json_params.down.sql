BEGIN;
ALTER TABLE metadata
    DROP COLUMN params;
END;