package main

import (
	"TP2/Database"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/notnil/chess"
	"github.com/notnil/chess/uci"
)

type GameResponse struct {
	gameUUID      string
	gameFEN       string
	playerList    string
	encryptionKey string
}

func (gr GameResponse) encode(serverKey string) []byte {
	fen, err := chess.FEN(gr.gameFEN)
	if err != nil {
		fmt.Println(err.Error())
	}
	accumulatedData := ""
	gameUUIDByte := BuildSubTLV(131, []byte(gr.gameUUID))
	keyByte := BuildSubTLV(4, []byte(gr.encryptionKey))

	if gr.gameFEN == "" {
		playerListByte := BuildSubTLV(133, []byte(gr.playerList))
		accumulatedData += gr.gameUUID + gr.playerList + gr.encryptionKey
		binary.Write(&gameUUIDByte, binary.BigEndian, playerListByte.Bytes())
	} else {
		gameFENByte := BuildSubTLV(132, []byte(chess.NewGame(fen).Position().Board().Draw()))
		accumulatedData += gr.gameUUID + chess.NewGame(fen).Position().Board().Draw() + gr.encryptionKey
		binary.Write(&gameUUIDByte, binary.BigEndian, gameFENByte.Bytes())
	}
	signatureByte := BuildSubTLV(3, []byte(SignMessage(serverKey, accumulatedData)))
	binary.Write(&gameUUIDByte, binary.BigEndian, keyByte.Bytes())
	binary.Write(&gameUUIDByte, binary.BigEndian, signatureByte.Bytes())
	return gameUUIDByte.Bytes()
}

type GameActionResponse struct {
	action       byte
	gameUUID     string
	gameFEN      string
	moveResponse string
	serverMove   string
	bestMove     string
	outcome      string
	turn         byte
	err          string
}

func (gar GameActionResponse) encode(serverKey string, encryptionKey string) []byte {
	response := new(bytes.Buffer)
	fen, err := chess.FEN(gar.gameFEN)
	if err != nil {
		fmt.Println(err.Error())
	}
	board := chess.NewGame(fen).Position().Board().Draw()

	action := BuildSubTLV(141, []byte{gar.action})
	gameUUIDByte := BuildSubTLV(131, []byte(gar.gameUUID))
	gameBoardByte := BuildSubTLV(132, []byte(board))
	signatureData := string(gar.action) + gar.gameUUID + board
	accumulatedData := action.String() + gameUUIDByte.String() + gameBoardByte.String()

	switch gar.action {
	case MOVE_RESPONSE, OPPONENT_MOVE_RESPONSE:
		moveResponseByte := BuildSubTLV(142, []byte(gar.moveResponse)) // à modifié
		accumulatedData += moveResponseByte.String()
		signatureData += gar.moveResponse
		if gar.serverMove != "" {
			serverMoveByte := BuildSubTLV(143, []byte(gar.serverMove))
			accumulatedData += serverMoveByte.String()
			signatureData += gar.serverMove
		} else {
			if gar.turn != UNDEFINED {
				turnByte := BuildSubTLV(134, []byte{gar.turn})
				accumulatedData += turnByte.String()
				signatureData += string(gar.turn)
			}
		}

	case GAME_OUTCOME:
		moveResponseByte := BuildSubTLV(142, []byte(gar.moveResponse))
		if gar.serverMove != "" {
			serverMoveByte := BuildSubTLV(142, []byte(gar.serverMove))
			accumulatedData += serverMoveByte.String()
			signatureData += gar.serverMove
		}
		outcomeByte := BuildSubTLV(144, []byte(gar.outcome))
		accumulatedData += outcomeByte.String() + moveResponseByte.String()
		signatureData += gar.outcome + gar.moveResponse
	case ERROR:
		errorByte := BuildSubTLV(199, []byte(gar.err))
		if gar.bestMove != "" {
			bestMoveByte := BuildSubTLV(145, []byte(gar.bestMove))
			accumulatedData += bestMoveByte.String()
			signatureData += gar.bestMove
		}
		accumulatedData += errorByte.String()
		signatureData += gar.err
	}
	signature := SignMessage(serverKey, signatureData)
	signatureByte := BuildSubTLV(3, []byte(signature))
	accumulatedData += signatureByte.String()
	accumulatedData, err = Encrypt(accumulatedData, encryptionKey)
	if err != nil {
		fmt.Println(err.Error())
	}
	binary.Write(response, binary.BigEndian, []byte(accumulatedData))
	return response.Bytes()
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go UDPServer(&wg)
	go TCPServer(&wg)
	wg.Wait()
}

func TCPServer(wg *sync.WaitGroup) {
	defer wg.Done()
	tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8000")
	if err != nil {
		fmt.Println("Erreur lors de l'écoute sur le port 8000: ", err)
		return
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println("Erreur lors de l'écoute sur le port 8000: ", err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
		}
		handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	var serverKey string
	var clientKey string
	for {
		buffer := make([]byte, 1024)
		_, err := conn.Read(buffer)
		if err != nil {
			fmt.Println(err)
			return
		}

		handleTLVTCP(conn, buffer, &serverKey, &clientKey)
	}
}

func handleTLVTCP(conn net.Conn, data []byte, serverKey *string, clientKey *string) {
	if len(data) < 3 {
		fmt.Println("Message trop court, ignoré.")
		return
	}

	tag := data[0]
	length := binary.BigEndian.Uint16(data[1:3])
	if int(length)+3 > len(data) {
		fmt.Println("Longueur invalide, message ignoré.")
		return
	}

	value := data[3 : 3+length]

	switch tag {
	case 0: // Authentification
		fmt.Println("> " + string(value))
		*serverKey = auth(string(value))
		*clientKey = strings.Split(string(value), "|")[1]
		secretResponse := BuildTLV(100, []byte(*serverKey))
		_, err := conn.Write(secretResponse)
		if err != nil {
			fmt.Println(err.Error())
		}

	case 1:
		splitedMessage := strings.Split(string(value), "|")
		if SignMessage(*clientKey, splitedMessage[0]) == splitedMessage[1] {
			fmt.Println("> " + string(value))
			data := getMenu(splitedMessage[2])

			message := BuildTLV(101, []byte(data+"|"+SignMessage(*serverKey, data)))
			_, err := conn.Write(message)
			if err != nil {
				fmt.Println(err.Error())
			}
		} else {
			fmt.Println("> Bad packet: " + string(value))

			message := BuildTLV(199, []byte("bad packet|"+SignMessage(*serverKey, "bad packet")))

			_, err := conn.Write(message)
			if err != nil {
				fmt.Println(err.Error())
			}
		}

	default:
		fmt.Println("Tag inconnu")
	}
}

func UDPServer(wg *sync.WaitGroup) {
	connectedClients := make(map[string]*net.UDPAddr)
	onGoingGame := make(map[string]string)
	defer wg.Done()
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8001")
	if err != nil {
		fmt.Println("Erreur lors de l'écoute sur le port 8001: ", err)
		return
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Erreur lors de l'écoute sur le port 8001: ", err)
		return
	}

	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		_, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println(err)
			return
		}

		handleTLVUDP(conn, addr, buffer, &connectedClients, &onGoingGame)
	}
}

func handleTLVUDP(conn *net.UDPConn, addr *net.UDPAddr, data []byte, connectedUsers *map[string]*net.UDPAddr, onGoingGame *map[string]string) {
	var serverKey string
	var clientKey string
	if len(data) < 3 {
		fmt.Println("Message trop court, ignoré.")
		return
	}

	tag := data[0]
	length := binary.BigEndian.Uint16(data[1:3])
	if int(length)+3 > len(data) {
		fmt.Println("Longueur invalide, message ignoré.")
		return
	}

	value := data[3 : 3+length]
	switch tag {
	case 0: // Authentification
		serverKey = auth(string(value))
		secretResponse := BuildTLV(100, []byte(serverKey))
		(*connectedUsers)[strings.Split(string(value), "|")[0]] = addr

		_, err := conn.WriteToUDP(secretResponse, addr)
		if err != nil {
			fmt.Println(err.Error())
		}

	case 1:
		splittedMessage := strings.Split(string(value), "|")
		serverKey, clientKey = getKeys(splittedMessage[2])
		if SignMessage(clientKey, splittedMessage[0]) == splittedMessage[1] {
			data := getMenu(splittedMessage[2])
			message := BuildTLV(101, []byte(data+"|"+SignMessage(serverKey, data)))

			_, err := conn.WriteToUDP(message, addr)
			if err != nil {
				fmt.Println(err.Error())
			}

		} else {
			fmt.Println("> Bad packet: " + string(value))
			message := BuildTLV(199, []byte("bad packet|"+SignMessage(serverKey, "bad packet")))

			_, err := conn.WriteToUDP(message, addr)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	case 30: // Jouer
		signature := ""
		accumulatedData := ""
		client := ""
		opponent := ""
		action := ""
		gameUUID := ""
		var gameType byte

		parseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {

			case 3: // Signature
				signature = string(subValue)
			case 10: // Action
				action = string(subValue)
				accumulatedData += action
			case 11: // GameType
				gameType = subValue[0]
			case 12: // Adversaire
				opponent = string(subValue)
				accumulatedData += opponent
			case 13: // Client
				client = string(subValue)
				accumulatedData += client
			case 14: // UUID partie
				gameUUID = string(subValue)
				accumulatedData += gameUUID
			default:

			}
		})
		serverKey, clientKey = getKeys(client)
		encryptionKey, err := GenerateKey()
		if err != nil {
			fmt.Println(err.Error())
		}
		if gameType == SOLO { // Jouer en solo
			gameUUID = createSoloGame(SOLO, client, encryptionKey)
			gr := GameResponse{
				gameUUID:      gameUUID,
				gameFEN:       getGame(gameUUID),
				encryptionKey: encryptionKey,
			}
			(*onGoingGame)[client] = gameUUID
			_, err := conn.WriteToUDP(BuildTLV(130, gr.encode(serverKey)), addr)
			if err != nil {
				fmt.Println(err.Error())
			}
		} else { // Jouer contre un autre joueur
			gr := GameResponse{
				gameUUID:      createSoloGame(PLAYER_VS_PLAYER, client, encryptionKey),
				playerList:    Database.GetAvailablePLayer(client),
				encryptionKey: encryptionKey,
			}
			(*onGoingGame)[client] = gr.gameUUID

			_, err := conn.WriteToUDP(BuildTLV(130, gr.encode(serverKey)), addr)
			if err != nil {
				fmt.Println(err.Error())
			}
		}

		if SignMessage(clientKey, accumulatedData) == signature {
			if action == "play" {
				if gameType == 1 {

				}
			}
		}
	case 35: // Rejoindre une partie
		signature := ""
		accumulatedData := ""
		client := ""
		gameUUID := ""

		parseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {

			case 3: // Signature
				signature = string(subValue)
			case 13: // Client
				client = string(subValue)
				accumulatedData += client
			case 14: // UUID partie
				gameUUID = string(subValue)
				accumulatedData += gameUUID
			default:

			}
		})

		serverKey, clientKey = getKeys(client)
		if SignMessage(clientKey, accumulatedData) == signature {
			encryptionKey := connectToGame(gameUUID, client)
			response := GameResponse{
				gameUUID:      gameUUID,
				gameFEN:       getGame(gameUUID),
				encryptionKey: encryptionKey,
			}

			(*onGoingGame)[client] = response.gameUUID
			_, err := conn.WriteToUDP(BuildTLV(130, response.encode(serverKey)), addr)
			if err != nil {
				fmt.Println(err.Error())
			}

			opponent := Database.GetUserName(Database.GetPlayerPId(response.gameUUID))
			opServerKey, _ := getKeys(opponent)
			gr := GameResponse{
				gameUUID:      response.gameUUID,
				gameFEN:       getGame(gameUUID),
				encryptionKey: Database.GetPlayerPKey(response.gameUUID),
			}
			_, err = conn.WriteToUDP(BuildTLV(130, gr.encode(opServerKey)), (*connectedUsers)[opponent])
			if err != nil {
				fmt.Println(err.Error())
			}
		} else {
			fmt.Println("Mauvaise signature join")
		}

	case 40: // Action dans la partie
		signature := ""
		var action byte
		gameUUID := ""
		move := ""
		client := ""
		accumulatedData := ""

		client, _ = mapkey((*connectedUsers), addr)
		gameUUID = (*onGoingGame)[client]
		key := getEncryptionKey(client, gameUUID)
		value, err := Decrypt(string(value), key)
		if err != nil {
			fmt.Println(err.Error())
		}
		parseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {
			case 3: // Signature
				signature += string(subValue)
			case 10: // Action
				action = subValue[0]
				accumulatedData += string(action)
			case 13: // Client
				accumulatedData += client
			case 41: // Déplacement
				move += string(subValue)
				accumulatedData += move
			case 42: // gameUUID
				accumulatedData += gameUUID
			}
		})

		serverKey, clientKey = getKeys(client)
		if SignMessage(clientKey, accumulatedData) == signature {
			moveError := ""
			var response GameActionResponse
			var serverMove string = ""
			switch action {
			case MOVE: // Action de jouer un coup
				newFen, bestMove, err := verifyMove(getGame(gameUUID), move, gameUUID)
				if err != nil {
					fmt.Println(err.Error())
					moveError = err.Error()
					serverMove = bestMove
				}

				isFinished, outcome := checkGameOutcome(newFen, gameUUID)
				turn := manageOpponentTurn(newFen)
				if Database.GetPlayerSId(gameUUID) != -1 {
					if err == nil {

						opponentResponse := prepareResponse(err, isFinished, gameUUID, bestMove, moveError, newFen, outcome, serverMove, OPPONENT_MOVE_RESPONSE, turn)
						serverMove = ""
						opponentName := Database.GetUserName(Database.GetPlayerSId(gameUUID))
						if opponentName == client {
							opponentName = Database.GetUserName(Database.GetPlayerPId(gameUUID))
						}
						opponentServerKey, _ := getKeys(opponentName)
						_, connErr := conn.WriteTo(BuildTLV(140, opponentResponse.encode(opponentServerKey, getEncryptionKey(opponentName, gameUUID))), (*connectedUsers)[opponentName])
						if connErr != nil {
							fmt.Println(connErr)
						}
					}
				} else if !isFinished && err == nil {
					newFen, serverMove = playServerTurn(newFen, gameUUID)
					isFinished, outcome = checkGameOutcome(newFen, gameUUID)
				}

				response = prepareResponse(err, isFinished, gameUUID, bestMove, moveError, newFen, outcome, serverMove, MOVE_RESPONSE, turn)
				_, connErr := conn.WriteToUDP(BuildTLV(140, response.encode(serverKey, getEncryptionKey(client, gameUUID))), addr)
				if connErr != nil {
					fmt.Println(err.Error())
				}

			default:
				fmt.Println("Mauvaise action")
			}
		} else {
			fmt.Println("Mauvaise signature")
		}

	case 98: // Quitter
		splittedMessage := strings.Split(string(value), "|")
		_, clientKey = getKeys(splittedMessage[2])
		if SignMessage(clientKey, splittedMessage[0]) == splittedMessage[1] {
			fmt.Println("> " + string(value))
			clientChangeStatus(splittedMessage[2], OFFLINE)
			delete(*connectedUsers, splittedMessage[2])
		}
	default:
		fmt.Println("Tag inconnu")
	}
}

func BuildSubTLV(tag byte, value []byte) bytes.Buffer {
	length := uint16(len(value))
	buffer := new(bytes.Buffer)
	buffer.WriteByte(tag)
	binary.Write(buffer, binary.BigEndian, length)
	buffer.Write(value)
	return *buffer
}

func parseSubTLV(data []byte, handleSubTLV func(byte, []byte)) {
	offset := 0
	for offset < len(data) {
		subTag := data[offset]
		subLength := binary.BigEndian.Uint16(data[offset+1 : offset+3])
		subValue := data[offset+3 : offset+3+int(subLength)]
		handleSubTLV(subTag, subValue)
		offset += 3 + int(subLength)
	}
}

func BuildTLV(tag byte, value []byte) []byte {
	length := uint16(len(value))
	buffer := make([]byte, 3+length)
	buffer[0] = tag
	binary.BigEndian.PutUint16(buffer[1:3], length)
	copy(buffer[3:], value)
	return buffer
}

func auth(client string) string {
	secret := generateUUID()

	clientName := strings.Split(client, "|")[0]

	clientKey := strings.Split(client, "|")[1]

	clientExist, err := Database.UserExist(clientName)
	if err != nil {
		fmt.Println(err)
	}

	if clientExist {
		Database.ChangeStatus(clientName, ONLINE)
		return Database.GetServerKey(clientName)
	} else {
		Database.InsertUser(clientName, ONLINE, secret, clientKey)
	}

	return secret
}

func getKeys(clientName string) (string, string) {
	serverKey := Database.GetServerKey(clientName)
	clientKey := Database.GetClientKey(clientName)
	return serverKey, clientKey
}

func generateUUID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Int31(), rand.Int31n(0xFFFF), rand.Int31n(0xFFFF),
		rand.Int31n(0xFFFF), rand.Int63n(0xFFFFFFFFFFFF))
}

func SignMessage(secretKey, message string) string {
	data := secretKey + message
	hash := sha256.New()
	hash.Write([]byte(data))
	signature := hash.Sum(nil)
	return fmt.Sprintf("%x", signature)
}

func clientChangeStatus(client string, status int) {
	Database.ChangeStatus(client, status)
}

func getMenu(name string) string {
	var menu string = "- play\n- join\n-Joueur vs joueur: \n"
	return menu + Database.GetAvailablePLayer(name)
}

func createSoloGame(gameType byte, client string, encryptionKey string) string {
	game := chess.NewGame()
	gameUUID := generateUUID()

	if gameType == SOLO {
		Database.InsertNewGame(game.FEN(), ONGOING, Database.GetUserId(client), gameUUID, encryptionKey)
	} else {
		Database.InsertNewGame(game.FEN(), WAITING, Database.GetUserId(client), gameUUID, encryptionKey)
	}

	return gameUUID
}

func connectToGame(gameUUID string, client string) string {
	key, _ := GenerateKey()
	Database.UpdateSecondaryPlayer(gameUUID, Database.GetUserId(client), key)
	return key
}

func getGame(gameUUID string) string {
	return Database.GetGameFen(gameUUID)
}

func verifyMove(fen string, move string, gameUUID string) (string, string, error) {
	formattedFEN, err := chess.FEN(fen)
	if err != nil {
		fmt.Println(err.Error())
		return fen, "", err
	}
	game := chess.NewGame(formattedFEN, chess.UseNotation(chess.UCINotation{}))

	if game.Outcome() != chess.NoOutcome {
		return fen, "", err
	}

	if err := game.MoveStr(move); err != nil {
		fmt.Println(err.Error())
		eng, errEng := uci.New("stockfish")
		if errEng != nil {
			return fen, "", err
		}
		engPos := uci.CmdPosition{Position: game.Position()}
		engGo := uci.CmdGo{SearchMoves: game.ValidMoves(), MoveTime: time.Second / 100}
		errEng = eng.Run(engPos, engGo)
		return fen, eng.SearchResults().BestMove.String(), err
	}
	Database.UpdateGame(game.FEN(), gameUUID)
	return game.FEN(), "", err
}

func playServerTurn(fen string, gameUUID string) (string, string) {
	formattedFen, err := chess.FEN(fen)
	if err != nil {
		fmt.Println(err.Error())
		return "", ""
	}
	eng, err := uci.New("stockfish")
	if err != nil {
		fmt.Println(err.Error())
		return "", ""
	}

	defer eng.Close()

	if err := eng.Run(uci.CmdUCI, uci.CmdIsReady, uci.CmdUCINewGame); err != nil {
		panic(err)
	}
	game := chess.NewGame(formattedFen, chess.UseNotation(chess.UCINotation{}))
	engPos := uci.CmdPosition{Position: game.Position()}
	engGo := uci.CmdGo{SearchMoves: game.ValidMoves(), MoveTime: time.Second / 100}

	if err := eng.Run(engPos, engGo); err != nil {
		fmt.Println(err.Error())
		return "", ""
	}

	move := eng.SearchResults().BestMove
	err = game.MoveStr(move.String())
	if err != nil {
		fmt.Println(err.Error())
		return "", ""
	}
	Database.UpdateGame(game.FEN(), gameUUID)
	return game.FEN(), move.String()
}

func manageOpponentTurn(newFen string) byte {
	fen, _ := chess.FEN(newFen)
	game := chess.NewGame(fen)
	switch game.Position().Turn().String() {
	case "b":
		fmt.Println("Black")
		return BLACK
	case "w":
		fmt.Println("White")
		return WHITE
	default:
		fmt.Println("WTF")
		return UNDEFINED
	}
}

func prepareResponse(err error, isFinished bool, gameUUID string, bestMove string, moveError string, newFen string, outcome string, serverMove string, action byte, turn byte) GameActionResponse {

	if err != nil { // Erreur
		return GameActionResponse{
			action:   ERROR,
			gameUUID: gameUUID,
			gameFEN:  getGame(gameUUID),
			bestMove: bestMove,
			err:      moveError,
		}
	} else if isFinished { // Partie terminée
		return GameActionResponse{
			action:       GAME_OUTCOME,
			gameUUID:     gameUUID,
			gameFEN:      newFen,
			moveResponse: outcome,
			serverMove:   serverMove,
			err:          moveError,
			outcome:      outcome,
		}
	} else { // Coup a été joué
		return GameActionResponse{
			action:       action,
			gameUUID:     gameUUID,
			gameFEN:      newFen,
			moveResponse: "Le coup à été joué",
			serverMove:   serverMove,
			err:          moveError,
			turn:         turn,
		}
	}
}

func checkGameOutcome(fen string, gameUUID string) (bool, string) {
	formattedFen, err := chess.FEN(fen)
	if err != nil {
		fmt.Println(err.Error())
		return false, err.Error()
	}

	game := chess.NewGame(formattedFen, chess.UseNotation(chess.UCINotation{}))

	if game.Outcome() == chess.NoOutcome {
		return false, ""
	}
	switch game.Outcome() {
	case chess.BlackWon:
		Database.UpdateGameStatus(FINISHED, gameUUID)
		return true, "Les noirs ont gagné par " + game.Method().String() + " !"
	case chess.WhiteWon:
		Database.UpdateGameStatus(FINISHED, gameUUID)
		return true, "Les blancs ont gagné par " + game.Method().String() + " !"
	case chess.Draw:
		Database.UpdateGameStatus(FINISHED, gameUUID)
		return true, "La partie est nulle !"
	}
	Database.UpdateGameStatus(FINISHED, gameUUID)
	return true, game.Outcome().String()
}

func getEncryptionKey(client string, gameUUID string) string {
	playerS := Database.GetPlayerSId(gameUUID)

	if playerS == Database.GetUserId(client) {
		return Database.GetPlayerSKey(gameUUID)
	}
	return Database.GetPlayerPKey(gameUUID)
}
