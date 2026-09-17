CREATE TABLE rolls (
    id        bigserial   PRIMARY KEY,
    guild_id  text        NOT NULL,
    user_id   text        NOT NULL,
    slug      text        NOT NULL,
    name      text        NOT NULL,
    kind      text        NOT NULL,
    cost      bigint      NOT NULL,
    rolled_at timestamptz NOT NULL
);

CREATE INDEX rolls_player_idx ON rolls (guild_id, user_id, rolled_at DESC);
