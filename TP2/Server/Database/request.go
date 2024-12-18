package Database

const (
	INSERT_USER          = `INSERT INTO users(name, status, key, clientKey) VALUES($name, $status, $key, $clientKey);`
	USER_EXISTS          = `SELECT clientKey FROM users WHERE name = $username;`
	GET_CLIENT_KEY       = `SELECT clientKey FROM users WHERE name = $username;`
	GET_SERVER_KEY       = `SELECT key FROM users WHERE name = $username;`
	CHANGE_STATUS        = `UPDATE users SET status = ? WHERE name = ?;`
	GET_AVAILABLE_PLAYER = `SELECT name FROM users WHERE status = 1 AND name != ?`
	GET_USER_ID_BY_NAME  = `SELECT id FROM users WHERE name = ?;`
	GET_USER_NAME_BY_ID  = `SELECT name FROM users WHERE id = ?;`

	INSERT_NEW_GAME         = `INSERT INTO games (fen, status, player_P, uuid, player_p_key) VALUES($fen, $status, $player_P, $uuid, $key);`
	GET_GAME_FEN            = `SELECT fen FROM games WHERE uuid = ? AND status != 3;`
	GET_PLAYER_P_KEY        = `SELECT player_p_key FROM games WHERE uuid = ?;`
	GET_PLAYER_S_KEY        = `SELECT player_s_key FROM games WHERE uuid = ?;`
	GET_PLAYER_P_ID         = `SELECT player_P FROM games WHERE uuid = ?;`
	GET_PLAYER_S_ID         = `SELECT player_S FROM games WHERE uuid = ?`
	UPDATE_SECONDARY_PLAYER = `UPDATE games SET player_S = ?, player_s_key = ? WHERE uuid = ?;`
	UPDATE_GAME_STATUS      = `UPDATE games SET status = ? WHERE uuid = ?;`
	UPDATE_FEN              = `UPDATE games SET fen = ? WHERE uuid = ?;`
	GET_PLAYER_GAMES        = `SELECT id, uuid FROM games WHERE player_P == $playerId OR player_S == $playerId AND status != 2 ORDER BY id DESC LIMIT 5;`
)
