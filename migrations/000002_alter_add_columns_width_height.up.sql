BEGIN;
ALTER TABLE metadata
    ADD COLUMN width  INT,
    ADD COLUMN height INT;
END;