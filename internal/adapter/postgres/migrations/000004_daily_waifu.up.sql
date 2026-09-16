CREATE TABLE daily_waifu (
    day           date        PRIMARY KEY,
    slug          text        NOT NULL,
    uuid          text        NOT NULL DEFAULT '',
    name          text        NOT NULL,
    original_name text        NOT NULL DEFAULT '',
    romaji_name   text        NOT NULL DEFAULT '',
    picture_url   text        NOT NULL DEFAULT '',
    likes         integer     NOT NULL DEFAULT 0,
    trash         integer     NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now()
);
