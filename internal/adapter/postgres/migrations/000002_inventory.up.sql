CREATE TABLE inventory (
    guild_id    text        NOT NULL,
    user_id     text        NOT NULL,
    slug        text        NOT NULL,
    uuid        text        NOT NULL DEFAULT '',
    name        text        NOT NULL,
    picture_url text        NOT NULL DEFAULT '',
    likes       integer     NOT NULL DEFAULT 0,
    trash       integer     NOT NULL DEFAULT 0,
    acquired_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, user_id, slug),
    FOREIGN KEY (guild_id, user_id) REFERENCES players (guild_id, user_id) ON DELETE CASCADE
);

CREATE INDEX inventory_name_prefix_idx ON inventory (guild_id, user_id, lower(name) text_pattern_ops);
