CREATE TABLE ranking_snapshots (
    id          bigserial   PRIMARY KEY,
    fetched_at  timestamptz NOT NULL,
    cutoff_page integer     NOT NULL,
    row_count   integer     NOT NULL,
    complete    boolean     NOT NULL DEFAULT false
);

CREATE TABLE ranking_rows (
    snapshot_id   bigint           NOT NULL REFERENCES ranking_snapshots (id) ON DELETE CASCADE,
    position      integer          NOT NULL,
    slug          text             NOT NULL,
    uuid          text             NOT NULL DEFAULT '',
    name          text             NOT NULL,
    original_name text             NOT NULL DEFAULT '',
    romaji_name   text             NOT NULL DEFAULT '',
    picture_url   text             NOT NULL DEFAULT '',
    likes         integer          NOT NULL,
    trash         integer          NOT NULL,
    score         double precision NOT NULL,
    stars         smallint         NOT NULL,
    PRIMARY KEY (snapshot_id, position)
);

CREATE INDEX ranking_rows_slug_idx ON ranking_rows (snapshot_id, slug);
