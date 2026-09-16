CREATE TABLE players (
    guild_id         text        NOT NULL,
    user_id          text        NOT NULL,
    coins            bigint      NOT NULL DEFAULT 200 CHECK (coins >= 0),
    daily_claimed_at timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, user_id)
);
