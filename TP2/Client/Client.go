package main

import (
	utils "TP2Client/Utils"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

type GameRequest struct {
	action   byte   ///< Byte signifiant l'action
	gameType byte   ///< Byte signifiant le type de partie
	client   string ///< Client
}

func (gr GameRequest) Encode(clientKey string) []byte {
	typeByte := buildTLV(11, []byte{gr.gameType})
	clientByte := buildTLV(13, []byte(gr.client))
	signatureByte := buildTLV(3, []byte(signMessage(clientKey, string(gr.action)+string(gr.gameType))))
	value := buildTLV(10, []byte(string(gr.action)))

	binary.Write(&value, binary.BigEndian, typeByte.Bytes())
	binary.Write(&value, binary.BigEndian, clientByte.Bytes())
	binary.Write(&value, binary.BigEndian, signatureByte.Bytes())
	return value.Bytes()
}

type GameJoinRequest struct {
	client   string ///< Client
	gameUUID string ///< Id de la partie
	action   byte
}

func (gjr GameJoinRequest) Encode(clientKey string) []byte {
	accumulatedData := gjr.client + gjr.gameUUID
	request := new(bytes.Buffer)
	clientByte := buildTLV(13, []byte(gjr.client))
	gameByte := buildTLV(14, []byte(gjr.gameUUID))
	if gjr.action != 0 {
		actionByte := buildTLV(53, []byte{gjr.action})
		binary.Write(request, binary.BigEndian, actionByte.Bytes())
	}

	signature := buildTLV(3, []byte(signMessage(clientKey, accumulatedData)))

	binary.Write(request, binary.BigEndian, clientByte.Bytes())
	binary.Write(request, binary.BigEndian, gameByte.Bytes())
	binary.Write(request, binary.BigEndian, signature.Bytes())
	return request.Bytes()
}

type GameTurn struct {
	action   byte   ///< Byte signifiant l'action
	move     string ///< Coup joué
	gameUUID string ///< Id de la partie
	client   string ///< Client
}

func (gt GameTurn) Encode(clientKey string, encryptionKey string) []byte {
	request := new(bytes.Buffer)
	signatureData := string(gt.action) + gt.move + gt.gameUUID + gt.client
	action := buildTLV(10, []byte{gt.action})
	moveByte := buildTLV(41, []byte(gt.move))
	gameByte := buildTLV(42, []byte(gt.gameUUID))
	clientByte := buildTLV(13, []byte(gt.client))
	signature := buildTLV(3, []byte(signMessage(clientKey, signatureData)))
	accumulatedData := action.String() + moveByte.String() + gameByte.String() + clientByte.String() + signature.String()
	accumulatedData, err := Encrypt(accumulatedData, encryptionKey)
	if err != nil {
		fmt.Println(err.Error())
	}

	binary.Write(request, binary.BigEndian, []byte(accumulatedData))
	return request.Bytes()
}

type LoadGame struct {
	client string
}

func (lg LoadGame) Encode(clientKey string) []byte {
	request := new(bytes.Buffer)
	signatureData := lg.client
	clientByte := buildTLV(13, []byte(lg.client))
	signature := buildTLV(3, []byte(signMessage(clientKey, signatureData)))
	accumulatedData := clientByte.String() + signature.String()
	binary.Write(request, binary.BigEndian, []byte(accumulatedData))
	return request.Bytes()
}

type Quit struct {
	Client string
}

func (q Quit) Encode(clientKey string) []byte {
	request := new(bytes.Buffer)
	signatureData := q.Client
	clientByte := buildTLV(13, []byte(q.Client))
	signature := buildTLV(3, []byte(signMessage(clientKey, signatureData)))
	accumulatedData := clientByte.String() + signature.String()
	binary.Write(request, binary.BigEndian, []byte(accumulatedData))
	return request.Bytes()
}

func main() {
	var serverKey string           ///< Clé de signature du serveur
	var clientKey string           ///< Clé de signature du client
	var encryptionKey string       ///< Clé de chiffrement
	var gameUUID string            ///< L'id de la partie
	var inGame bool = false        ///< Booléen signifiant si le client est dans une partie
	var waiting bool = false       ///< Booléen signifiant si le client est en attente
	var color byte = UNDEFINED     ///< Couleur du client dans la partie
	var colorTurn byte = UNDEFINED ///< Couleur qui doit jouer dans la partie
	var message string = ""        ///< Message à envoyer
	var soloGame bool = true
	var responseOk bool = false
	var loadOldGame bool = false
	var isUp bool = true

	if len(os.Args) < 4 {
		fmt.Println("Please provide protocol host:port to connect to and username")
		os.Exit(1)
	}

	// Connect to the address
	conn, err := net.Dial(os.Args[1], "127.0.0.1:"+os.Args[2])

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Authentification
	newClient, clientKey := readFile(os.Args[3])

	if newClient {
		clientKey = generateUUID()
		writeConfig(os.Args[3], string(clientKey))
	}

	sendTLV(conn, 0, []byte(os.Args[3]+"|"+string(clientKey)))
	messageReader := bufio.NewReader(os.Stdin)

	buffer := make([]byte, 1024)

	for isUp {
		message = ""
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println(err)
			return
		}

		tag, value := parseTLV(buffer[:n])
		handleTLV(tag, value, &serverKey, &encryptionKey, &gameUUID, &inGame, &waiting, &color, &colorTurn, &loadOldGame, &soloGame)
		if !soloGame && isUp {
			if waiting || color != colorTurn { // Multijoueur seulement
				n, err = conn.Read(buffer)
				if err != nil {
					fmt.Println(err)
					return
				}
				tag, value := parseTLV(buffer[:n])
				handleTLV(tag, value, &serverKey, &encryptionKey, &gameUUID, &inGame, &waiting, &color, &colorTurn, &loadOldGame, &soloGame)
			}

			if color == colorTurn {
				message = utils.ReadConsole(messageReader)
			}
		} else {

			message = utils.ReadConsole(messageReader)

		}

		message = strings.ReplaceAll(message, "\n", "")

		if loadOldGame {
			request := GameJoinRequest{
				client:   os.Args[3],
				gameUUID: message,
				action:   LOAD_GAME,
			}
			loadOldGame = false
			soloGame = false
			sendTLV(conn, 35, request.Encode(clientKey))
		} else if inGame && message != "" && isUp {
			gt := GameTurn{
				action:   MOVE,
				move:     message,
				gameUUID: gameUUID,
				client:   os.Args[3],
			}
			sendTLV(conn, 40, gt.Encode(clientKey, encryptionKey))
		} else if message != "" {
			commandOk := false

			for !commandOk {
				switch message {
				case "play":
					commandOk = true
					for !responseOk {
						fmt.Print("Écrivez le numéro correspondant à votre choix: \n1 - Solo\n2 - Multijoueur\n> ")
						gameType, _ := messageReader.ReadString('\n')
						gameType = strings.ReplaceAll(gameType, "\n", "")
						switch gameType {
						case "1":
							soloGame = true
							responseOk = true
							gr := GameRequest{
								action:   PLAY,
								gameType: SOLO,
								client:   os.Args[3],
							}
							sendTLV(conn, 30, gr.Encode(clientKey))
						case "2":
							soloGame = false
							responseOk = true
							gr := GameRequest{
								action:   PLAY,
								gameType: PLAYER_VS_PLAYER,
								client:   os.Args[3],
							}
							sendTLV(conn, 30, gr.Encode(clientKey))
						default:
							fmt.Println("Le choix: " + gameType + " n'est pas reconnu")
						}
					}
				case "join":
					commandOk = true
					responseOk := false
					for !responseOk {
						fmt.Println("Que voulez-vous faire ?\n1 - Rejoindre quelqu'un\n2 - Charger une ancienne partie")
						joinChoice := utils.ReadConsole(messageReader)
						switch joinChoice {
						case "1":
							responseOk = true
							fmt.Print("Entrez le UUID de la partie > ")
							gameId, _ := messageReader.ReadString('\n')
							gameId = strings.ReplaceAll(gameId, "\n", "")
							soloGame = false
							request := GameJoinRequest{
								client:   os.Args[3],
								gameUUID: gameId,
								action:   0,
							}
							sendTLV(conn, 35, request.Encode(clientKey))
						case "2":
							responseOk = true
							lg := LoadGame{
								client: os.Args[3],
							}
							sendTLV(conn, 50, lg.Encode(clientKey))
						default:
							fmt.Println("Le choix: " + joinChoice + " n'est pas reconnu")
						}
					}
				case "exit":
					commandOk = true
					request := Quit{
						Client: os.Args[3],
					}
					isUp = false
					sendTLV(conn, 98, request.Encode(clientKey))
				default:
					fmt.Println("Commande inconnu")
					message = utils.ReadConsole(messageReader)
				}
			}
		}
	}
}

func buildTLV(tag byte, value []byte) bytes.Buffer {
	length := uint16(len(value))
	buffer := new(bytes.Buffer)
	buffer.WriteByte(tag)
	binary.Write(buffer, binary.BigEndian, length)
	buffer.Write(value)
	return *buffer
}

func sendTLV(conn net.Conn, tag byte, value []byte) {
	length := uint16(len(value))
	buffer := new(bytes.Buffer)
	buffer.WriteByte(tag)
	binary.Write(buffer, binary.BigEndian, length)
	buffer.Write(value)
	_, err := conn.Write(buffer.Bytes())
	if err != nil {
		fmt.Println(err)
		return
	}
}

func parseTLV(data []byte) (byte, []byte) {
	if len(data) < 3 {
		return 0, nil
	}
	tag := data[0]
	length := binary.BigEndian.Uint16(data[1:3])
	if int(length)+3 > len(data) {
		return 0, nil
	}
	return tag, data[3 : 3+length]
}

func handleTLV(tag byte, data []byte, serverKey *string, encryptionKey *string, gameUUID *string, inGame *bool, waiting *bool, color *byte, colorTurn *byte, loadOldGame *bool, soloGame *bool) {
	switch tag {
	case 100: // Auth
		*serverKey = string(data)
		fmt.Println("Écrivez l'action voulu: \nplay\njoin\nexit")

	case 101: // Action
		splitedMessage := strings.Split(string(data), "|")
		if signMessage(*serverKey, splitedMessage[0]) == splitedMessage[1] {
			fmt.Println("> " + splitedMessage[0])
		} else {
			fmt.Println("bad packet")
		}
	case 130: // Réponse d'une partie
		signature := ""
		accumulatedData := ""
		board := ""
		playList := ""
		var responseColor byte
		var turnResponse byte

		parseSubTLV(data, func(subTag byte, subValue []byte) {
			switch subTag {

			case 3: // Signature
				signature = string(subValue)
			case 4: // Encryption key
				*encryptionKey = string(subValue)
				accumulatedData += *encryptionKey
			case 131: // Game UUID
				*gameUUID = string(subValue)
				accumulatedData += *gameUUID
			case 132: // FEN
				board = string(subValue)
				accumulatedData += board
			case 133: // Player list
				playList = string(subValue)
				accumulatedData += playList
			case 134: // Couleur
				responseColor = subValue[0]
				accumulatedData += string(responseColor)
			case 135: // Tour
				turnResponse = subValue[0]
				accumulatedData += string(turnResponse)
			default:

			}
		})

		if signMessage(*serverKey, accumulatedData) == signature {
			if board == "" {
				fmt.Print("Voici l'id de la partie, envoyez la à votre ami !: ")
				fmt.Println(*gameUUID)
				*waiting = true
				*color = WHITE
			} else {
				if responseColor == UNDEFINED {
					if *color == UNDEFINED {
						*color = BLACK
					}
					*colorTurn = WHITE
					fmt.Println(board)
					*inGame = true
					*waiting = false
					*soloGame = true
				} else {
					*color = responseColor
					*colorTurn = turnResponse
					fmt.Println(board)
					*inGame = true
					*waiting = false
				}
			}
		}

	case 140: // Move response
		signature := ""
		accumulatedData := ""
		board := ""
		moveResponse := ""
		serverMove := ""
		var action byte
		gameErr := ""
		outcome := ""
		bestMove := ""

		data, err := Decrypt(string(data), *encryptionKey)
		if err != nil {
			fmt.Println(err.Error())
		}

		parseSubTLV([]byte(data), func(subTag byte, subValue []byte) {
			switch subTag {

			case 3: // Signature
				signature = string(subValue)
			case 131: // gameUUID
				*gameUUID = string(subValue)
				accumulatedData += *gameUUID
			case 132: // board
				board = string(subValue)
				accumulatedData += board
			case 134: // Tour
				*colorTurn = subValue[0]
				accumulatedData += string(*colorTurn)
			case 141: // Action
				action = subValue[0]
				accumulatedData += string(action)
			case 142: // move response
				moveResponse = string(subValue)
				accumulatedData += moveResponse
			case 143: // serverMode
				serverMove = string(subValue)
				accumulatedData += serverMove
			case 144: // outcome
				outcome = string(subValue)
				accumulatedData += outcome
			case 145: // Best move
				bestMove = string(subValue)
				accumulatedData += bestMove
			case 199: // erreur
				gameErr = string(subValue)
				accumulatedData += gameErr
			}
		})
		if signMessage(*serverKey, accumulatedData) == signature {

			if gameErr != "" {
				fmt.Println("Erreur: " + gameErr)
				if bestMove != "" {
					fmt.Println("Le meilleur coup jouable est: " + bestMove)
					fmt.Println(board)
				}
			} else if outcome != "" {
				fmt.Println("LA PARTIE EST TERMINÉE")
				fmt.Println(board)
				fmt.Println("La partie est terminé: " + outcome)
				*inGame = false
			} else {
				if serverMove != "" {
					fmt.Println("L'adversaire à joué: " + serverMove)
				}
				fmt.Println(board)
				if *color == *colorTurn {
					fmt.Print("Entrez votre coup: ")
				}
			}
		} else {
			fmt.Println("Bad packet")
		}
	case 150:
		signature := ""
		list := ""
		err := ""
		accumulatedData := ""

		parseSubTLV(data, func(subTag byte, subValue []byte) {
			switch subTag {
			case 3:
				signature = string(subValue)
			case 151:
				list = string(subValue)
				accumulatedData += string(subValue)
			case 152:
				err = string(subValue)
				accumulatedData += string(subValue)
			}
		})
		if signMessage(*serverKey, accumulatedData) == signature {
			if err != "" {
				*loadOldGame = false
				fmt.Println("Erreur big")
				fmt.Println(err)
			} else {
				*loadOldGame = true
				splittedList := strings.Split(list, ",")
				fmt.Println("Entrez le uuid de la partie pour la charger")
				for i := 0; i < len(splittedList); i++ {
					fmt.Println(splittedList[i])
				}
			}
		}

	case 199: // Mauvais packet
		fmt.Println("> Votre requête n'a pas pu être effectuer")
	}
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

func signMessage(secretKey, message string) string {
	data := secretKey + message
	hash := sha256.New()
	hash.Write([]byte(data))
	signature := hash.Sum(nil)
	return fmt.Sprintf("%x", signature)
}

func generateUUID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Int31(), rand.Int31n(0xFFFF), rand.Int31n(0xFFFF),
		rand.Int31n(0xFFFF), rand.Int63n(0xFFFFFFFFFFFF))
}

func tagManager(message string) byte {
	switch message {
	case "exit", "EXIT":
		return 98
	case "play", "PLAY":
		return 30
	default:
		return 99
	}
}

func readFile(name string) (bool, string) {
	file, err := os.Open("config" + strings.ReplaceAll(name, " ", "_") + ".txt")
	if err != nil {
		return true, ""
	}
	defer file.Close()

	reader := bufio.NewScanner(file)
	for reader.Scan() {
		if strings.Contains(reader.Text(), name) {
			key := strings.Split(reader.Text(), ":")[1]
			return false, key
		}
	}
	return true, ""
}

func writeConfig(name string, key string) {
	f, err := os.Create("config" + strings.ReplaceAll(name, " ", "_") + ".txt")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = f.WriteString(name + ":" + key + "\n")
	if err != nil {
		fmt.Println(err)
		f.Close()
		return
	}
	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
}
