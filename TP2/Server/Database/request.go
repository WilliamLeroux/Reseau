package Database

const (
	INSERT_USER          = `INSERT INTO users(name, status, key, clientKey) VALUES($name, $status, $key, $clientKey);`
	USER_EXISTS          = `SELECT clientKey FROM users WHERE name = $username;`
	GET_CLIENT_KEY       = `SELECT clientKey FROM users WHERE name = $username;`
	GET_SERVER_KEY       = `SELECT key FROM users WHERE name = $username;`
	CHANGE_STATUS        = `UPDATE users SET status = ? WHERE name = ?;`
	GET_AVAILABLE_PLAYER = `SELECT name FROM users WHERE status = 1 AND name != ?`
	GET_USER_ID_BY_NAME  = `SELECT id FROM users WHERE name = ?;`

	INSERT_NEW_GAME         = `INSERT INTO games (fen, status, player_P, uuid, player_p_key) VALUES($fen, $status, $player_P, $uuid, $key);`
	UPDATE_SECONDARY_PLAYER = `UPDATE games SET player_S = ?, player_s_key WHERE uuid = ?;`
	GET_GAME_FEN            = `SELECT fen FROM games WHERE uuid = ? AND status != 3;`
	UPDATE_FEN              = `UPDATE games SET fen = ? WHERE uuid = ?;`
	GET_PLAYER_P_KEY        = `SELECT player_p_key FROM games WHERE uuid = ?;`
	GET_PLAYER_S_KEY        = `SELECT player_s_key FROM games WHERE uuid = ?;`
)
