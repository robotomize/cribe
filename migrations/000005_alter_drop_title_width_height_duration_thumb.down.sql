BEGIN;
ALTER TABLE metadata
    ADD COLUMN duration INT,
    ADD COLUMN thumb    TEXT,
    ADD COLUMN title    TEXT,
    ADD COLUMN width    INT,
    ADD COLUMN height   INT;
END;