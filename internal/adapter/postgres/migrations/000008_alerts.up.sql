CREATE TABLE alert_settings (
    guild_id   text        NOT NULL,
    user_id    text        NOT NULL,
    enabled    boolean     NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, user_id)
);

CREATE TABLE alert_dm_closed (
    user_id   text        PRIMARY KEY,
    closed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE alerts_sent (
    event_id text        NOT NULL,
    user_id  text        NOT NULL,
    sent_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)
);

CREATE INDEX favorites_kind_slug_idx ON favorites (kind, slug);
