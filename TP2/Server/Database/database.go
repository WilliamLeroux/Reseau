package Database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func dbCreation() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./Database/chessDB.sqlite")
	return db, err
}

func InsertUser(name string, status int, key string, clientKey string) {
	db, err := dbCreation()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	tx, _ := db.Begin()

	query, err := db.Prepare(INSERT_USER)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}

	_, err = query.Exec(name, status, key, clientKey)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_ = tx.Commit()
}

func UserExist(name string) (bool, error) {

	db, err := dbCreation()

	if err != nil {
		return false, err
	}
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	query, err := db.Prepare(USER_EXISTS)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}

	result, err := query.Query(name)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if result != nil {
		var clientKey string = ""
		for result.Next() {
			result.Scan(&clientKey)
		}
		if clientKey == "" {
			return false, err
		}
	}
	return true, nil
}

func GetClientKey(name string) string {
	var clientKey string = ""
	db, err := dbCreation()

	if err != nil {
		return clientKey
	}
	tx, err := db.Begin()
	if err != nil {
		return clientKey
	}
	query, err := db.Prepare(GET_CLIENT_KEY)
	if err != nil {
		_ = tx.Rollback()
		return clientKey
	}

	result, err := query.Query(name)
	if err != nil {
		_ = tx.Rollback()
		return clientKey
	}
	if result != nil {

		for result.Next() {
			result.Scan(&clientKey)
		}
	}
	return clientKey
}

func GetServerKey(name string) string {
	var serverKey string = ""
	db, err := dbCreation()

	if err != nil {
		return serverKey
	}
	tx, err := db.Begin()
	if err != nil {
		return serverKey
	}
	query, err := db.Prepare(GET_SERVER_KEY)
	if err != nil {
		_ = tx.Rollback()
		return serverKey
	}

	result, err := query.Query(name)
	if err != nil {
		_ = tx.Rollback()
		return serverKey
	}
	if result != nil {

		for result.Next() {
			result.Scan(&serverKey)
		}
	}
	return serverKey
}

func ChangeStatus(name string, status int) {
	db, err := dbCreation()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	tx, _ := db.Begin()

	query, err := db.Prepare(CHANGE_STATUS)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_, err = query.Exec(status, name)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_ = tx.Commit()
}

func GetAvailablePLayer(name string) string {
	var playerList string = ""
	var tempPlayer string
	db, err := dbCreation()

	if err != nil {
		return playerList
	}
	tx, err := db.Begin()
	if err != nil {
		return playerList
	}
	query, err := db.Prepare(GET_AVAILABLE_PLAYER)
	if err != nil {
		_ = tx.Rollback()
		return playerList
	}

	result, err := query.Query(name)
	if err != nil {
		_ = tx.Rollback()
		return playerList
	}
	if result != nil {

		for result.Next() {
			result.Scan(&tempPlayer)
			playerList = playerList + tempPlayer + "\n"
		}
	}
	return playerList
}

func GetUserId(client string) int {
	id := -1
	db, err := dbCreation()

	if err != nil {
		return id
	}
	tx, err := db.Begin()
	if err != nil {
		return id
	}
	query, err := db.Prepare(GET_USER_ID_BY_NAME)
	if err != nil {
		_ = tx.Rollback()
		return id
	}

	result, err := query.Query(client)
	if err != nil {
		_ = tx.Rollback()
		return id
	}
	if result != nil {

		for result.Next() {
			result.Scan(&id)

		}
	}
	return id
}

func InsertNewGame(fen string, status int, player_P int, uuid string, key string) {
	db, err := dbCreation()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	tx, _ := db.Begin()

	query, err := db.Prepare(INSERT_NEW_GAME)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_, err = query.Exec(fen, status, player_P, uuid, key)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_ = tx.Commit()
}

func UpdateSecondaryPlayer(gameUUID string, client int) {
	db, err := dbCreation()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	tx, _ := db.Begin()

	query, err := db.Prepare(UPDATE_SECONDARY_PLAYER)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_, err = query.Exec(client, gameUUID)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_ = tx.Commit()
}

func GetGameFen(gameUUID string) string {
	fen := ""
	db, err := dbCreation()

	if err != nil {
		return fen
	}
	tx, err := db.Begin()
	if err != nil {
		return fen
	}
	query, err := db.Prepare(GET_GAME_FEN)
	if err != nil {
		_ = tx.Rollback()
		return fen
	}

	result, err := query.Query(gameUUID)
	if err != nil {
		_ = tx.Rollback()
		return fen
	}
	if result != nil {

		for result.Next() {
			result.Scan(&fen)

		}
	}
	return fen
}

func UpdateGame(fen string, uuid string) {
	db, err := dbCreation()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	tx, _ := db.Begin()

	query, err := db.Prepare(UPDATE_FEN)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_, err = query.Exec(fen, uuid)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_ = tx.Commit()
}

func GetPlayerPId(gameUUID string) int {
	id := -1
	db, err := dbCreation()

	if err != nil {
		return id
	}
	tx, err := db.Begin()
	if err != nil {
		return id
	}
	query, err := db.Prepare(GET_PLAYER_P_ID)
	if err != nil {
		_ = tx.Rollback()
		return id
	}

	result, err := query.Query(gameUUID)
	if err != nil {
		_ = tx.Rollback()
		return id
	}
	if result != nil {

		for result.Next() {
			result.Scan(&id)

		}
	}
	return id
}

func GetPlayerSId(gameUUID string) int {
	id := -1
	db, err := dbCreation()

	if err != nil {
		return id
	}
	tx, err := db.Begin()
	if err != nil {
		return id
	}
	query, err := db.Prepare(GET_PLAYER_S_ID)
	if err != nil {
		_ = tx.Rollback()
		return id
	}

	result, err := query.Query(gameUUID)
	if err != nil {
		_ = tx.Rollback()
		return id
	}
	if result != nil {

		for result.Next() {
			result.Scan(&id)

		}
	}
	return id
}

func GetPlayerPKey(gameUUID string) string {
	key := ""
	db, err := dbCreation()

	if err != nil {
		return key
	}
	tx, err := db.Begin()
	if err != nil {
		return key
	}
	query, err := db.Prepare(GET_PLAYER_P_KEY)
	if err != nil {
		_ = tx.Rollback()
		return key
	}

	result, err := query.Query(gameUUID)
	if err != nil {
		_ = tx.Rollback()
		return key
	}
	if result != nil {

		for result.Next() {
			result.Scan(&key)

		}
	}
	return key
}

func GetPlayerSKey(gameUUID string) string {
	key := ""
	db, err := dbCreation()

	if err != nil {
		return key
	}
	tx, err := db.Begin()
	if err != nil {
		return key
	}
	query, err := db.Prepare(GET_PLAYER_S_KEY)
	if err != nil {
		_ = tx.Rollback()
		return key
	}

	result, err := query.Query(gameUUID)
	if err != nil {
		_ = tx.Rollback()
		return key
	}
	if result != nil {

		for result.Next() {
			result.Scan(&key)

		}
	}
	return key
}

func UpdateGameStatus(status int, gameUUID string) {
	db, err := dbCreation()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	tx, _ := db.Begin()

	query, err := db.Prepare(UPDATE_GAME_STATUS)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_, err = query.Exec(status, gameUUID)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err.Error())
		return
	}
	_ = tx.Commit()
}
