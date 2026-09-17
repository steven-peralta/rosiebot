CREATE TABLE banners (
    week_start  timestamptz PRIMARY KEY,
    series_slug text        NOT NULL,
    series_uuid text        NOT NULL DEFAULT '',
    series_name text        NOT NULL,
    series_url  text        NOT NULL DEFAULT '',
    picture_url text        NOT NULL DEFAULT '',
    description text        NOT NULL DEFAULT '',
    characters  jsonb       NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
