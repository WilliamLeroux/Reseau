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
	gameUUID string
	gameFEN  string
}

func (gr GameResponse) encode(serverKey string) []byte {
	fen, err := chess.FEN(gr.gameFEN)
	if err != nil {
		fmt.Println(err.Error())
	}
	gameUUIDByte := buildSubTLV(131, []byte(gr.gameUUID))

	gameFENByte := buildSubTLV(132, []byte(chess.NewGame(fen).Position().Board().Draw()))
	signatureByte := buildSubTLV(3, []byte(signMessage(serverKey, gr.gameUUID+chess.NewGame(fen).Position().Board().Draw())))

	binary.Write(&gameUUIDByte, binary.BigEndian, gameFENByte.Bytes())
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
	err          string
}

func (gar GameActionResponse) encode(serverKey string) []byte {
	fen, err := chess.FEN(gar.gameFEN)
	if err != nil {
		fmt.Println(err.Error())
	}
	board := chess.NewGame(fen).Position().Board().Draw()
	accumulatedData := string(gar.action) + gar.gameUUID + board

	value := buildSubTLV(141, []byte{gar.action})
	gameUUIDByte := buildSubTLV(131, []byte(gar.gameUUID))
	gameBoardByte := buildSubTLV(132, []byte(board))
	binary.Write(&value, binary.BigEndian, gameUUIDByte.Bytes())
	binary.Write(&value, binary.BigEndian, gameBoardByte.Bytes())

	switch gar.action {
	case MOVE_RESPONSE:
		moveResponseByte := buildSubTLV(142, []byte(gar.moveResponse))
		serverMoveByte := buildSubTLV(143, []byte(gar.serverMove))
		binary.Write(&value, binary.BigEndian, moveResponseByte.Bytes())
		binary.Write(&value, binary.BigEndian, serverMoveByte.Bytes())
		accumulatedData += gar.moveResponse + gar.serverMove
	case GAME_OUTCOME:
		moveResponseByte := buildSubTLV(142, []byte(gar.moveResponse))
		if gar.serverMove != "" {
			serverMoveByte := buildSubTLV(142, []byte(gar.serverMove))
			binary.Write(&value, binary.BigEndian, serverMoveByte.Bytes())
			accumulatedData += gar.serverMove
		}
		outcomeByte := buildSubTLV(144, []byte(gar.outcome))
		binary.Write(&value, binary.BigEndian, outcomeByte.Bytes())
		binary.Write(&value, binary.BigEndian, moveResponseByte.Bytes())
		accumulatedData += gar.outcome
		accumulatedData += gar.moveResponse
	case ERROR:
		errorByte := buildSubTLV(199, []byte(gar.err))
		if gar.bestMove != "" {
			bestMoveByte := buildSubTLV(145, []byte(gar.bestMove))
			binary.Write(&value, binary.BigEndian, bestMoveByte.Bytes())
			accumulatedData += gar.bestMove
		}
		binary.Write(&value, binary.BigEndian, errorByte.Bytes())
		accumulatedData += gar.err
	}
	signature := signMessage(serverKey, accumulatedData)
	signatureByte := buildSubTLV(3, []byte(signature))
	binary.Write(&value, binary.BigEndian, signatureByte.Bytes())

	return value.Bytes()
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
		secretResponse := buildTLV(100, []byte(*serverKey))
		_, err := conn.Write(secretResponse)
		if err != nil {
			fmt.Println(err.Error())
		}

	case 1:
		splitedMessage := strings.Split(string(value), "|")
		if signMessage(*clientKey, splitedMessage[0]) == splitedMessage[1] {
			fmt.Println("> " + string(value))
			data := getMenu(splitedMessage[2])

			message := buildTLV(101, []byte(data+"|"+signMessage(*serverKey, data)))
			_, err := conn.Write(message)
			if err != nil {
				fmt.Println(err.Error())
			}
		} else {
			fmt.Println("> Bad packet: " + string(value))

			message := buildTLV(199, []byte("bad packet|"+signMessage(*serverKey, "bad packet")))

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

		handleTLVUDP(conn, addr, buffer, &connectedClients)
	}
}

func handleTLVUDP(conn *net.UDPConn, addr *net.UDPAddr, data []byte, connectedUsers *map[string]*net.UDPAddr) {
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
		fmt.Println("> " + string(value))
		serverKey = auth(string(value))
		secretResponse := buildTLV(100, []byte(serverKey))
		(*connectedUsers)[strings.Split(string(value), "|")[0]] = addr

		_, err := conn.WriteToUDP(secretResponse, addr)
		if err != nil {
			fmt.Println(err.Error())
		}

	case 1:
		splittedMessage := strings.Split(string(value), "|")
		serverKey, clientKey = getKeys(splittedMessage[2])
		if signMessage(clientKey, splittedMessage[0]) == splittedMessage[1] {
			fmt.Println("> " + string(value))
			data := getMenu(splittedMessage[2])
			message := buildTLV(101, []byte(data+"|"+signMessage(serverKey, data)))

			_, err := conn.WriteToUDP(message, addr)
			if err != nil {
				fmt.Println(err.Error())
			}

		} else {
			fmt.Println("> Bad packet: " + string(value))
			message := buildTLV(199, []byte("bad packet|"+signMessage(serverKey, "bad packet")))

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

		parseSubTLV(value, func(subTag byte, subValue []byte) {
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
		if gameType == SOLO {
			gr := GameResponse{
				gameUUID: createSoloGame("solo", client),
				gameFEN:  getGame(gameUUID),
			}
			_, err := conn.WriteToUDP(buildTLV(130, gr.encode(serverKey)), addr)
			if err != nil {
				fmt.Println(err.Error())
			}
		}

		if signMessage(clientKey, accumulatedData) == signature {
			if action == "play" {
				if gameType == 1 {

				}
			}
		}
	case 40: // Action dans la partie
		signature := ""
		var action byte
		gameUUID := ""
		move := ""
		client := ""
		accumulatedData := ""

		parseSubTLV(value, func(subTag byte, subValue []byte) {
			switch subTag {
			case 3: // Signature
				signature += string(subValue)
			case 10: // Action
				action = subValue[0]
				accumulatedData += string(action)
			case 13: // Client
				client += string(subValue)
				accumulatedData += client
			case 41: // Déplacement
				move += string(subValue)
				accumulatedData += move
			case 42: // gameUUID
				gameUUID += string(subValue)
				accumulatedData += gameUUID
			}
		})

		serverKey, clientKey = getKeys(client)
		if signMessage(clientKey, accumulatedData) == signature {
			moveError := ""
			var response GameActionResponse
			var serverMove string = ""
			switch action {
			case MOVE:
				newFen, bestMove, err := verifyMove(getGame(gameUUID), move, gameUUID)
				if err != nil {
					fmt.Println(err.Error())
					moveError = err.Error()
					serverMove = bestMove
				}

				isFinished, outcome := checkGameOutcome(newFen, gameUUID)

				if !isFinished && err == nil {
					newFen, serverMove = playServerTurn(newFen, gameUUID)
					isFinished, outcome = checkGameOutcome(newFen, gameUUID)
				}

				if err != nil {
					response = GameActionResponse{
						action:   ERROR,
						gameUUID: gameUUID,
						gameFEN:  getGame(gameUUID),
						bestMove: bestMove,
						err:      moveError,
					}
				} else if isFinished {
					response = GameActionResponse{
						action:       GAME_OUTCOME,
						gameUUID:     gameUUID,
						gameFEN:      newFen,
						moveResponse: outcome,
						serverMove:   serverMove,
						err:          moveError,
						outcome:      outcome,
					}
				} else {
					response = GameActionResponse{
						action:       MOVE_RESPONSE,
						gameUUID:     gameUUID,
						gameFEN:      newFen,
						moveResponse: "Le coup à été joué",
						serverMove:   serverMove,
						err:          moveError,
					}
				}
				_, err = conn.WriteToUDP(buildTLV(140, response.encode(serverKey)), addr)
				if err != nil {
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
		if signMessage(clientKey, splittedMessage[0]) == splittedMessage[1] {
			fmt.Println("> " + string(value))
			clientChangeStatus(splittedMessage[2], OFFLINE)
			delete(*connectedUsers, splittedMessage[2])
		}
	default:
		fmt.Println("Tag inconnu")
	}
}

func buildSubTLV(tag byte, value []byte) bytes.Buffer {
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

func buildTLV(tag byte, value []byte) []byte {
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

func signMessage(secretKey, message string) string {
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

func createSoloGame(gameType string, client string) string {
	game := chess.NewGame()
	gameUUID := generateUUID()
	encryptionKey, err := GenerateKey()
	if err != nil {
		fmt.Println(err.Error())
	}

	if gameType == "solo" {
		Database.InsertNewGame(game.FEN(), ONGOING, Database.GetUserId(client), gameUUID, encryptionKey)
	} else {
		Database.InsertNewGame(game.FEN(), WAITING, Database.GetUserId(client), gameUUID, encryptionKey)
	}

	return gameUUID
}

func connectToGame(gameUUID string, client string) { // Ajouter encryption
	Database.UpdateSecondaryPlayer(gameUUID, Database.GetUserId(client))
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
		return true, "Les noirs ont gagné par " + game.Method().String() + " !"
	case chess.WhiteWon:
		return true, "Les blancs ont gagné par " + game.Method().String() + " !"
	case chess.Draw:
		return true, "La partie est nulle !"
	}
	return true, game.Outcome().String()
}
