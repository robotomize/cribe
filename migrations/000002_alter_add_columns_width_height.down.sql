BEGIN;
ALTER TABLE metadata
    DROP COLUMN width,
    DROP COLUMN height;
END;