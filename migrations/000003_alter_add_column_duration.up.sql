BEGIN;
ALTER TABLE metadata
    ADD COLUMN duration INT;
END;