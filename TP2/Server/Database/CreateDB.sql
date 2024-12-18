-- SQLite
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    name TEXT NOT NULL,
    status INTEGER NOT NULL,
    key TEXT NOT NULL,
    clientKey TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS games (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    fen TEXT NOT NULL,
    status INTEGER NOT NULL,
    player_P INTEGER NOT NULL,
    player_S INTEGER DEFAULT -1,
    uuid TEXT NOT NULL,
    player_p_key TEXT,
    player_s_key TEXT
);