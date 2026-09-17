CREATE TABLE favorites (
    guild_id    text        NOT NULL,
    user_id     text        NOT NULL,
    kind        text        NOT NULL CHECK (kind IN ('waifu', 'series')),
    slug        text        NOT NULL,
    name        text        NOT NULL,
    url         text        NOT NULL DEFAULT '',
    picture_url text        NOT NULL DEFAULT '',
    added_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, user_id, kind, slug)
);
