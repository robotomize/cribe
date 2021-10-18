BEGIN;
ALTER TABLE metadata
    DROP COLUMN duration,
    DROP COLUMN thumb,
    DROP COLUMN title,
    DROP COLUMN width,
    DROP COLUMN height;
END;