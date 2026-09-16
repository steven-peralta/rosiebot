CREATE TABLE waifu_cache (
    slug         text        PRIMARY KEY,
    payload      jsonb       NOT NULL,
    fetched_at   timestamptz NOT NULL,
    last_read_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX waifu_cache_last_read_idx ON waifu_cache (last_read_at);

CREATE TABLE page_cache (
    key        text        PRIMARY KEY,
    payload    jsonb       NOT NULL,
    fetched_at timestamptz NOT NULL
);

CREATE INDEX page_cache_fetched_idx ON page_cache (fetched_at);
