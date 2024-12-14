package main

import (
	"TP2/Database"
	"TP2/Model"

	utils "TP2/Utils"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/notnil/chess"
	"github.com/notnil/chess/uci"
)

func main() {
	var wg sync.WaitGroup
	onGoingGame := make(map[string]string)
	udpConnectedUser := make(map[string]*net.UDPAddr)
	tcpConnectedUser := make(map[string]*net.Conn)
	wg.Add(2)
	go UDPServer(&wg, &onGoingGame, &udpConnectedUser, &tcpConnectedUser)
	go TCPServer(&wg, &onGoingGame, &udpConnectedUser, &tcpConnectedUser)
	wg.Wait()
}

func TCPServer(wg *sync.WaitGroup, onGoingGame *map[string]string, udpConnectedClients *map[string]*net.UDPAddr, tcpConnectedClients *map[string]*net.Conn) {
	defer wg.Done()
	tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("Erreur lors de l'écoute sur le port 8080: ", err)
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
		go handleConnection(conn, onGoingGame, udpConnectedClients, tcpConnectedClients)
	}
}

func handleConnection(conn net.Conn, onGoingGame *map[string]string, udpConnectedClients *map[string]*net.UDPAddr, tcpConnectedClients *map[string]*net.Conn) {
	defer conn.Close()
	var serverKey string
	var clientKey string
	var encryptionKey string
	var gameUUID string
	var client string
	var isConnected bool = true

	for isConnected {
		buffer := make([]byte, 1024)
		_, err := conn.Read(buffer)
		if err != nil {
			fmt.Println(err)
			return
		}

		handleTLVTCP(conn, buffer, &client, &serverKey, &clientKey, &gameUUID, &encryptionKey, udpConnectedClients, tcpConnectedClients, onGoingGame, &isConnected)
	}
}

func handleTLVTCP(conn net.Conn, data []byte, client *string, serverKey *string, clientKey *string, gameUUID *string, encryptionKey *string, udpConnectedUsers *map[string]*net.UDPAddr, tcpConnectedCLients *map[string]*net.Conn, onGoingGame *map[string]string, isConnected *bool) {
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
		*serverKey = auth(string(value))
		_, *clientKey = getKeys(strings.Split(string(value), "|")[0])
		secretResponse := utils.BuildTLV(100, []byte(*serverKey))
		(*tcpConnectedCLients)[strings.Split(string(value), "|")[0]] = &conn
		*client = strings.Split(string(value), "|")[0]
		_, err := conn.Write(secretResponse)
		if err != nil {
			fmt.Println(err.Error())
		}

	case 1: // A supprimer
		splittedMessage := strings.Split(string(value), "|")
		if utils.SignMessage(*clientKey, splittedMessage[0]) == splittedMessage[1] {
			data := getMenu(splittedMessage[2])
			message := utils.BuildTLV(101, []byte(data+"|"+utils.SignMessage(*serverKey, data)))

			_, err := conn.Write(message)
			if err != nil {
				fmt.Println(err.Error())
			}

		} else {
			fmt.Println("> Bad packet: " + string(value))
			message := utils.BuildTLV(199, []byte("bad packet|"+utils.SignMessage(*serverKey, "bad packet")))

			_, err := conn.Write(message)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	case 30: // Jouer
		signature := ""
		accumulatedData := ""
		opponent := ""
		action := ""
		var gameType byte

		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
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
				*client = string(subValue)
				accumulatedData += *client
			case 14: // UUID partie
				*gameUUID = string(subValue)
				accumulatedData += *gameUUID
			default:

			}
		})
		*encryptionKey, _ = utils.GenerateKey()

		if gameType == Model.SOLO { // Jouer en solo
			*gameUUID = createSoloGame(Model.SOLO, *client, *encryptionKey)
			gr := Model.GameResponse{
				GameUUID:      *gameUUID,
				GameFEN:       getGame(*gameUUID),
				EncryptionKey: *encryptionKey,
			}
			(*onGoingGame)[*client] = *gameUUID
			_, err := conn.Write(utils.BuildTLV(130, gr.Encode(*serverKey)))
			if err != nil {
				fmt.Println(err.Error())
			}
		} else { // Jouer contre un autre joueur
			gr := Model.GameResponse{
				GameUUID:      createSoloGame(Model.PLAYER_VS_PLAYER, *client, *encryptionKey),
				PlayerList:    Database.GetAvailablePLayer(*client),
				EncryptionKey: *encryptionKey,
			}
			*gameUUID = gr.GameUUID
			(*onGoingGame)[*client] = gr.GameUUID

			_, err := conn.Write(utils.BuildTLV(130, gr.Encode(*serverKey)))
			if err != nil {
				fmt.Println(err.Error())
			}
		}

		if utils.SignMessage(*clientKey, accumulatedData) == signature {
			if action == "play" {
				if gameType == 1 {

				}
			}
		}
	case 35: // Rejoindre une partie
		signature := ""
		accumulatedData := ""
		var action byte = 0
		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {

			case 3: // Signature
				signature = string(subValue)
			case 13: // Client
				*client = string(subValue)
				accumulatedData += *client
			case 14: // UUID partie
				*gameUUID = string(subValue)
				accumulatedData += *gameUUID
			case 53: // Charger
				action = subValue[0]
			default:

			}
		})

		if utils.SignMessage(*clientKey, accumulatedData) == signature {
			*encryptionKey = connectToGame(*gameUUID, *client)
			if action == Model.LOAD_GAME {
				color := Model.UNDEFINED
				opponentName := Database.GetUserName(Database.GetPlayerSId(*gameUUID))
				if opponentName != "" {
					if opponentName == *client {
						color = Model.BLACK
					} else {
						color = Model.WHITE
					}
				}
				gr := Model.GameResponse{
					GameUUID:      *gameUUID,
					GameFEN:       getGame(*gameUUID),
					EncryptionKey: *encryptionKey,
					Color:         color,
				}
				_, err := conn.Write(utils.BuildTLV(130, gr.Encode(*serverKey)))
				if err != nil {
					fmt.Println(err.Error())
				}
			} else {
				response := Model.GameResponse{
					GameUUID:      *gameUUID,
					GameFEN:       getGame(*gameUUID),
					EncryptionKey: *encryptionKey,
					Color:         Model.UNDEFINED,
				}

				(*onGoingGame)[*client] = response.GameUUID
				_, err := conn.Write(utils.BuildTLV(130, response.Encode(*serverKey)))
				if err != nil {
					fmt.Println(err.Error())
				}

				opponent := Database.GetUserName(Database.GetPlayerPId(response.GameUUID))
				opServerKey, _ := getKeys(opponent)
				gr := Model.GameResponse{
					GameUUID:      response.GameUUID,
					GameFEN:       getGame(*gameUUID),
					EncryptionKey: Database.GetPlayerPKey(response.GameUUID),
					Color:         Model.UNDEFINED,
				}
				opponentUDPAddr := (*udpConnectedUsers)[opponent]
				var opponentTCPAddr net.Conn
				if opponentUDPAddr == nil {
					opponentTCPAddr = *(*tcpConnectedCLients)[opponent]
					_, err = opponentTCPAddr.Write(utils.BuildTLV(130, gr.Encode(opServerKey)))
				} else {
					_, err = conn.Write(utils.BuildTLV(130, gr.Encode(opServerKey))) // UDP
				}

				if err != nil {
					fmt.Println(err.Error())
				}
			}

		} else {
			fmt.Println("Mauvaise signature join")
		}

	case 40: // Action dans la partie
		signature := ""
		var action byte
		move := ""
		accumulatedData := ""

		value, err := utils.Decrypt(string(value), *encryptionKey)
		if err != nil {
			fmt.Println("[ERROR] " + err.Error())
		}

		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {
			case 3: // Signature
				signature += string(subValue)
			case 10: // Action
				action = subValue[0]
				accumulatedData += string(action)
			case 13: // Client
				accumulatedData += string(subValue)
			case 41: // Déplacement
				move += string(subValue)
				accumulatedData += move
			case 42: // gameUUID
				accumulatedData += string(subValue)
			}
		})

		if utils.SignMessage(*clientKey, accumulatedData) == signature {
			moveError := ""
			var response Model.GameActionResponse
			var serverMove string = ""
			switch action {
			case Model.MOVE: // Action de jouer un coup
				newFen, bestMove, err := verifyMove(getGame(*gameUUID), move, *gameUUID)
				if err != nil {
					fmt.Println(err.Error())
					if err.Error() != "chess: fen invalid notiation  must have 6 sections" {
						moveError = err.Error()
						serverMove = bestMove
					} else {
						err = nil
					}
				}
				isFinished, outcome := checkGameOutcome(newFen, *gameUUID)
				turn := manageOpponentTurn(newFen)
				if Database.GetPlayerSId(*gameUUID) != -1 {
					if err == nil {

						opponentResponse := prepareResponse(err, isFinished, *gameUUID, bestMove, moveError, newFen, outcome, serverMove, Model.OPPONENT_MOVE_RESPONSE, turn)
						serverMove = ""
						opponentName := Database.GetUserName(Database.GetPlayerSId(*gameUUID))
						if opponentName == *client {
							opponentName = Database.GetUserName(Database.GetPlayerPId(*gameUUID))
						}
						opponentServerKey, _ := getKeys(opponentName)
						opponentUDPAddr := (*udpConnectedUsers)[opponentName]
						var opponentTCPAddr net.Conn
						if opponentUDPAddr == nil {
							opponentTCPAddr = *(*tcpConnectedCLients)[opponentName]
							if opponentTCPAddr != nil {
								_, connErr := opponentTCPAddr.Write(utils.BuildTLV(140, opponentResponse.Encode(opponentServerKey, getEncryptionKey(opponentName, *gameUUID))))
								if connErr != nil {
									fmt.Println(connErr)
								}
							}

						} else {
							_, connErr := conn.Write(utils.BuildTLV(140, opponentResponse.Encode(opponentServerKey, getEncryptionKey(opponentName, *gameUUID)))) // UDP
							if connErr != nil {
								fmt.Println(connErr)
							}
						}
					}
				} else if !isFinished && err == nil {
					newFen, serverMove = playServerTurn(newFen, *gameUUID)
					isFinished, outcome = checkGameOutcome(newFen, *gameUUID)
				}

				response = prepareResponse(err, isFinished, *gameUUID, bestMove, moveError, newFen, outcome, serverMove, Model.MOVE_RESPONSE, turn)
				_, connErr := conn.Write(utils.BuildTLV(140, response.Encode(*serverKey, *encryptionKey)))
				if connErr != nil {
					fmt.Println(connErr.Error())
				}

			default:
				fmt.Println("Mauvaise action")
			}
		} else {
			fmt.Println("Mauvaise signature action")
		}

	case 50: // Charger une partie
		signature := ""
		accumulatedData := ""

		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {
			case 3: // Signature
				signature += string(subValue)
			case 13: // Client
				accumulatedData += string(subValue)
			}
		})

		if utils.SignMessage(*clientKey, accumulatedData) == signature {
			gameList := Database.GetPlayerGames(uint(Database.GetUserId(*client)))
			err := ""
			if gameList == "" {
				err = "Le joueur n'a pas de partie"
			}
			response := Model.GameListResponse{
				List:  gameList,
				Error: err,
			}
			_, connErr := conn.Write(utils.BuildTLV(150, response.Encode(*serverKey)))
			if connErr != nil {
				fmt.Println(connErr.Error())
			}
		}

	case 98: // Quitter
		signature := ""
		accumulatedData := ""

		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {
			case 3:
				signature += string(subValue)
			case 13:
				accumulatedData += string(subValue)
			}
			if utils.SignMessage(*clientKey, accumulatedData) == signature {
				clientChangeStatus(*client, Model.OFFLINE)
				delete(*tcpConnectedCLients, *client)
				RemoveUserFromGame(*client, onGoingGame)
				*isConnected = false
			}
		})
	default:
		fmt.Println("Tag inconnu")
	}
}

func UDPServer(wg *sync.WaitGroup, onGoingGame *map[string]string, udpConnectedClients *map[string]*net.UDPAddr, tcpConnectedClients *map[string]*net.Conn) {

	defer wg.Done()
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8081")
	if err != nil {
		fmt.Println("Erreur lors de l'écoute sur le port 8081: ", err)
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

		handleTLVUDP(conn, addr, buffer, udpConnectedClients, tcpConnectedClients, onGoingGame)
	}
}

func handleTLVUDP(conn *net.UDPConn, addr *net.UDPAddr, data []byte, udpConnectedUsers *map[string]*net.UDPAddr, tcpConnectedCLients *map[string]*net.Conn, onGoingGame *map[string]string) {
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
		secretResponse := utils.BuildTLV(100, []byte(serverKey))
		(*udpConnectedUsers)[strings.Split(string(value), "|")[0]] = addr
		_, err := conn.WriteToUDP(secretResponse, addr)
		if err != nil {
			fmt.Println(err.Error())
		}

	case 1:
		splittedMessage := strings.Split(string(value), "|")
		serverKey, clientKey = getKeys(splittedMessage[2])
		if utils.SignMessage(clientKey, splittedMessage[0]) == splittedMessage[1] {
			data := getMenu(splittedMessage[2])
			message := utils.BuildTLV(101, []byte(data+"|"+utils.SignMessage(serverKey, data)))

			_, err := conn.WriteToUDP(message, addr)
			if err != nil {
				fmt.Println(err.Error())
			}

		} else {
			fmt.Println("> Bad packet: " + string(value))
			message := utils.BuildTLV(199, []byte("bad packet|"+utils.SignMessage(serverKey, "bad packet")))

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

		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
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
		encryptionKey, err := utils.GenerateKey()
		if err != nil {
			fmt.Println(err.Error())
		}
		if gameType == Model.SOLO { // Jouer en solo
			gameUUID = createSoloGame(Model.SOLO, client, encryptionKey)
			gr := Model.GameResponse{
				GameUUID:      gameUUID,
				GameFEN:       getGame(gameUUID),
				EncryptionKey: encryptionKey,
			}
			(*onGoingGame)[client] = gameUUID
			_, err := conn.WriteToUDP(utils.BuildTLV(130, gr.Encode(serverKey)), addr)
			if err != nil {
				fmt.Println(err.Error())
			}
		} else { // Jouer contre un autre joueur
			gr := Model.GameResponse{
				GameUUID:      createSoloGame(Model.PLAYER_VS_PLAYER, client, encryptionKey),
				PlayerList:    Database.GetAvailablePLayer(client),
				EncryptionKey: encryptionKey,
			}
			(*onGoingGame)[client] = gr.GameUUID

			_, err := conn.WriteToUDP(utils.BuildTLV(130, gr.Encode(serverKey)), addr)
			if err != nil {
				fmt.Println(err.Error())
			}
		}

		if utils.SignMessage(clientKey, accumulatedData) == signature {
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
		var action byte = 0

		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {

			case 3: // Signature
				signature = string(subValue)
			case 13: // Client
				client = string(subValue)
				accumulatedData += client
			case 14: // UUID partie
				gameUUID = string(subValue)
				accumulatedData += gameUUID
			case 53: // Charger
				action = subValue[0]
			default:

			}
		})

		serverKey, clientKey = getKeys(client)
		if utils.SignMessage(clientKey, accumulatedData) == signature {
			encryptionKey := connectToGame(gameUUID, client)

			if action == Model.LOAD_GAME {
				color := Model.UNDEFINED
				opponentName := Database.GetUserName(Database.GetPlayerSId(gameUUID))
				if opponentName != "" {
					if opponentName == client {
						color = Model.BLACK
					} else {
						color = Model.WHITE
					}
				}
				gr := Model.GameResponse{
					GameUUID:      gameUUID,
					GameFEN:       getGame(gameUUID),
					EncryptionKey: encryptionKey,
					Color:         color,
				}
				_, err := conn.WriteToUDP(utils.BuildTLV(130, gr.Encode(serverKey)), addr)
				if err != nil {
					fmt.Println(err.Error())
				}
			} else {
				response := Model.GameResponse{
					GameUUID:      gameUUID,
					GameFEN:       getGame(gameUUID),
					EncryptionKey: encryptionKey,
					Color:         Model.UNDEFINED,
				}
				(*onGoingGame)[client] = response.GameUUID
				_, err := conn.WriteToUDP(utils.BuildTLV(130, response.Encode(serverKey)), addr)
				if err != nil {
					fmt.Println(err.Error())
				}

				opponent := Database.GetUserName(Database.GetPlayerPId(response.GameUUID))
				opServerKey, _ := getKeys(opponent)
				gr := Model.GameResponse{
					GameUUID:      response.GameUUID,
					GameFEN:       getGame(gameUUID),
					EncryptionKey: Database.GetPlayerPKey(response.GameUUID),
					Color:         Model.UNDEFINED,
				}
				opponentUDPAddr := (*udpConnectedUsers)[opponent]
				var opponentTCPAddr net.Conn
				if opponentUDPAddr == nil {
					opponentTCPAddr = *(*tcpConnectedCLients)[opponent]
					_, err = opponentTCPAddr.Write(utils.BuildTLV(130, gr.Encode(opServerKey)))
				} else {
					_, err = conn.WriteTo(utils.BuildTLV(130, gr.Encode(opServerKey)), opponentUDPAddr)
				}

				if err != nil {
					fmt.Println(err.Error())
				}
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
		client, _ = utils.Mapkey((*udpConnectedUsers), addr)
		gameUUID = (*onGoingGame)[client]
		key := getEncryptionKey(client, gameUUID)
		value, err := utils.Decrypt(string(value), key)
		if err != nil {
			fmt.Println(err.Error())
		}
		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
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
		if utils.SignMessage(clientKey, accumulatedData) == signature {
			moveError := ""
			var response Model.GameActionResponse
			var serverMove string = ""
			switch action {
			case Model.MOVE: // Action de jouer un coup
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

						opponentResponse := prepareResponse(err, isFinished, gameUUID, bestMove, moveError, newFen, outcome, serverMove, Model.OPPONENT_MOVE_RESPONSE, turn)
						serverMove = ""
						opponentName := Database.GetUserName(Database.GetPlayerSId(gameUUID))
						if opponentName == client {
							opponentName = Database.GetUserName(Database.GetPlayerPId(gameUUID))
						}
						opponentServerKey, _ := getKeys(opponentName)
						_, connErr := conn.WriteTo(utils.BuildTLV(140, opponentResponse.Encode(opponentServerKey, getEncryptionKey(opponentName, gameUUID))), (*udpConnectedUsers)[opponentName])
						if connErr != nil {
							fmt.Println(connErr)
						}
					}
				} else if !isFinished && err == nil {
					newFen, serverMove = playServerTurn(newFen, gameUUID)
					isFinished, outcome = checkGameOutcome(newFen, gameUUID)
				}

				response = prepareResponse(err, isFinished, gameUUID, bestMove, moveError, newFen, outcome, serverMove, Model.MOVE_RESPONSE, turn)
				_, connErr := conn.WriteToUDP(utils.BuildTLV(140, response.Encode(serverKey, getEncryptionKey(client, gameUUID))), addr)
				if connErr != nil {
					fmt.Println(err.Error())
				}

			default:
				fmt.Println("Mauvaise action")
			}
		} else {
			fmt.Println("Mauvaise signature")
		}
	case 50: // Charger une partie
		signature := ""
		accumulatedData := ""
		client := ""

		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {
			case 3: // Signature
				signature += string(subValue)
			case 13: // Client
				client = string(subValue)
				accumulatedData += string(subValue)
			}
		})
		serverKey, clientKey = getKeys(client)
		if utils.SignMessage(clientKey, accumulatedData) == signature {
			gameList := Database.GetPlayerGames(uint(Database.GetUserId(client)))
			err := ""
			if gameList == "" {
				err = "Le joueur n'a pas de partie"
			}
			response := Model.GameListResponse{
				List:  gameList,
				Error: err,
			}
			_, connErr := conn.WriteToUDP(utils.BuildTLV(150, response.Encode(serverKey)), addr)
			if connErr != nil {
				fmt.Println(connErr.Error())
			}
		}

	case 98: // Quitter
		signature := ""
		accumulatedData := ""
		client := ""

		utils.ParseSubTLV([]byte(value), func(subTag byte, subValue []byte) {
			switch subTag {
			case 3:
				signature += string(subValue)
			case 13:
				client = string(subValue)
				accumulatedData += string(subValue)
			}
			_, clientKey = getKeys(client)
			if utils.SignMessage(clientKey, accumulatedData) == signature {
				clientChangeStatus(client, Model.OFFLINE)
				delete(*udpConnectedUsers, client)
				RemoveUserFromGame(client, onGoingGame)
			}
		})
	default:
		fmt.Println("Tag inconnu")
	}
}

func RemoveUserFromGame(client string, onGoingGame *map[string]string) {
	delete(*onGoingGame, client)
}

func auth(client string) string {
	secret := utils.GenerateUUID()

	clientName := strings.Split(client, "|")[0]

	clientKey := strings.Split(client, "|")[1]

	clientExist, err := Database.UserExist(clientName)
	if err != nil {
		fmt.Println(err)
	}

	if clientExist {
		Database.ChangeStatus(clientName, Model.ONLINE)
		return Database.GetServerKey(clientName)
	} else {
		Database.InsertUser(clientName, Model.ONLINE, secret, clientKey)
	}

	return secret
}

func getKeys(clientName string) (string, string) {
	serverKey := Database.GetServerKey(clientName)
	clientKey := Database.GetClientKey(clientName)
	return serverKey, clientKey
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
	gameUUID := utils.GenerateUUID()

	if gameType == Model.SOLO {
		Database.InsertNewGame(game.FEN(), Model.ONGOING, Database.GetUserId(client), gameUUID, encryptionKey)
	} else {
		Database.InsertNewGame(game.FEN(), Model.WAITING, Database.GetUserId(client), gameUUID, encryptionKey)
	}

	return gameUUID
}

func connectToGame(gameUUID string, client string) string {
	id := Database.GetUserId(client)
	if id == Database.GetPlayerPId(gameUUID) {
		Database.UpdateGameStatus(Model.ONGOING, gameUUID)
		return Database.GetPlayerPKey(gameUUID)
	} else if id == Database.GetPlayerSId(gameUUID) {
		Database.UpdateGameStatus(Model.ONGOING, gameUUID)
		return Database.GetPlayerSKey(gameUUID)
	} else {
		key, _ := utils.GenerateKey()
		Database.UpdateSecondaryPlayer(gameUUID, Database.GetUserId(client), key)
		Database.UpdateGameStatus(Model.ONGOING, gameUUID)
		return key
	}
}

func getGame(gameUUID string) string {
	return Database.GetGameFen(gameUUID)
}

func verifyMove(fen string, move string, gameUUID string) (string, string, error) {
	formattedFEN, err := chess.FEN(fen)
	if err != nil {
		if err.Error() != "chess: fen invalid notiation  must have 6 sections" {
			return fen, "", err
		}
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
		_ = eng.Run(engPos, engGo)
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
		return Model.BLACK
	case "w":
		return Model.WHITE
	default:
		return Model.UNDEFINED
	}
}

func prepareResponse(err error, isFinished bool, gameUUID string, bestMove string, moveError string, newFen string, outcome string, serverMove string, action byte, turn byte) Model.GameActionResponse {

	if err != nil { // Erreur
		return Model.GameActionResponse{
			Action:   Model.ERROR,
			GameUUID: gameUUID,
			GameFEN:  getGame(gameUUID),
			BestMove: bestMove,
			Err:      moveError,
		}
	} else if isFinished { // Partie terminée
		return Model.GameActionResponse{
			Action:       Model.GAME_OUTCOME,
			GameUUID:     gameUUID,
			GameFEN:      newFen,
			MoveResponse: outcome,
			ServerMove:   serverMove,
			Err:          moveError,
			Outcome:      outcome,
		}
	} else { // Coup a été joué
		return Model.GameActionResponse{
			Action:       action,
			GameUUID:     gameUUID,
			GameFEN:      newFen,
			MoveResponse: "Le coup à été joué",
			ServerMove:   serverMove,
			Err:          moveError,
			Turn:         turn,
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
		Database.UpdateGameStatus(Model.FINISHED, gameUUID)
		return true, "Les noirs ont gagné par " + game.Method().String() + " !"
	case chess.WhiteWon:
		Database.UpdateGameStatus(Model.FINISHED, gameUUID)
		return true, "Les blancs ont gagné par " + game.Method().String() + " !"
	case chess.Draw:
		Database.UpdateGameStatus(Model.FINISHED, gameUUID)
		return true, "La partie est nulle !"
	}
	Database.UpdateGameStatus(Model.FINISHED, gameUUID)
	return true, game.Outcome().String()
}

func getEncryptionKey(client string, gameUUID string) string {
	playerS := Database.GetPlayerSId(gameUUID)

	if playerS == Database.GetUserId(client) {
		return Database.GetPlayerSKey(gameUUID)
	}
	return Database.GetPlayerPKey(gameUUID)
}
