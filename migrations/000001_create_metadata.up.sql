BEGIN;
CREATE TABLE IF NOT EXISTS metadata
(
    CONSTRAINT pk PRIMARY KEY (video_id, mime, quality),

    video_id   TEXT,
    title      TEXT,
    file_id    TEXT,
    mime       TEXT,
    quality    TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

END;