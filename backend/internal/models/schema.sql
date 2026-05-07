CREATE TABLE IF NOT EXISTS sessions (
	token TEXT PRIMARY KEY,
	data BLOB NOT NULL,
	expiry REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry);

CREATE TABLE IF NOT EXISTS live_matches (
    match_id INTEGER PRIMARY KEY AUTOINCREMENT, 
    white_player_id INTEGER NOT NULL, 
    black_player_id INTEGER NOT NULL,
    last_move_piece INTEGER,
    last_move_move INTEGER,
    current_fen TEXT DEFAULT 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1' NOT NULL,
    time_format_in_milliseconds INTEGER NOT NULL,
    increment_in_milliseconds INTEGER NOT NULL,
    white_player_time_remaining_in_milliseconds INTEGER NOT NULL,
    black_player_time_remaining_in_milliseconds INTEGER NOT NULL,
    game_history_json_string BLOB NOT NULL,
    unix_ms_time_of_last_move INTEGER NOT NULL,
    average_elo REAL NOT NULL,
    white_player_elo INTEGER NOT NULL,
    black_player_elo INTEGER NOT NULL,
    match_start_time INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS past_matches (
    match_id INTEGER PRIMARY KEY NOT NULL, 
    white_player_id INTEGER NOT NULL, 
    black_player_id INTEGER NOT NULL,
    last_move_piece INTEGER NOT NULL,
    last_move_move INTEGER NOT NULL,
    final_fen TEXT NOT NULL,
    time_format_in_milliseconds INTEGER NOT NULL,
    increment_in_milliseconds INTEGER NOT NULL,
    game_history_json_string BLOB NOT NULL,
    result INTEGER NOT NULL,
    result_reason INTEGER NOT NULL,
    white_player_elo INTEGER NOT NULL,
    black_player_elo INTEGER NOT NULL,
    white_player_elo_gain INTEGER NOT NULL,
    black_player_elo_gain INTEGER NOT NULL,
    average_elo REAL NOT NULL,
    match_start_time INTEGER NOT NULL,
    match_end_time INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    player_id INTEGER PRIMARY KEY NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    email TEXT,
    join_date INTEGER DEFAULT (strftime('%s', 'now')),
    last_seen INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE TABLE IF NOT EXISTS user_ratings (
    player_id INTEGER PRIMARY KEY NOT NULL,
    username TEXT UNIQUE NOT NULL,
    bullet_rating INTEGER DEFAULT 1500,
    blitz_rating INTEGER DEFAULT 1500,
    rapid_rating INTEGER DEFAULT 1500,
    classical_rating INTEGER DEFAULT 1500
);

CREATE INDEX IF NOT EXISTS past_matches_players_idx ON past_matches (white_player_id, black_player_id);