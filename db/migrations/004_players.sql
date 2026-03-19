-- +goose Up
CREATE TABLE players (
    vrp_id          INT             NOT NULL,
    server          TEXT            NOT NULL,
    player_name     TEXT            NOT NULL,
    first_seen      TIMESTAMPTZ     DEFAULT NOW(),
    last_seen       TIMESTAMPTZ     DEFAULT NOW(),
    PRIMARY KEY (vrp_id, server)
);

-- +goose Down
DROP TABLE players;
