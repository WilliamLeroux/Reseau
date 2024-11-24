package main

import (
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
	action   byte
	gameType byte
	client   string
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

type GameTurn struct {
	action   byte
	move     string
	gameUUID string
	client   string
}

func (gt GameTurn) Encode(clientKey string, encryptionKey string) []byte {
	request := new(bytes.Buffer)
	action := buildTLV(10, []byte{gt.action})
	moveByte := buildTLV(41, []byte(gt.move))
	gameByte := buildTLV(42, []byte(gt.gameUUID))
	clientByte := buildTLV(13, []byte(gt.client))
	signature := buildTLV(3, []byte(signMessage(clientKey, string(gt.action)+gt.move+gt.gameUUID+gt.client)))
	fmt.Println(signMessage(clientKey, string(gt.action)+gt.move+gt.gameUUID+gt.client))
	accumulatedData := action.String() + moveByte.String() + gameByte.String() + clientByte.String() + signature.String()
	accumulatedData, err := Encrypt(accumulatedData, encryptionKey)
	if err != nil {
		fmt.Println(err.Error())
	}

	binary.Write(request, binary.BigEndian, []byte(accumulatedData))
	return request.Bytes()
}

func main() {
	var serverKey string
	var clientKey string
	var encryptionKey string
	var gameUUID string
	var inGame bool = false
	var t byte = 1

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

	for t != 98 {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println(err)
			return
		}

		tag, value := parseTLV(buffer[:n])
		handleTLV(tag, value, &serverKey, &encryptionKey, &gameUUID, &inGame)
		fmt.Print("> ")
		message, err := messageReader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		message = strings.ReplaceAll(message, "\n", "")

		t = tagManager(message)

		messageSigned := signMessage(string(clientKey), message)

		if os.Args[1] == "udp" {
			messageSigned = messageSigned + "|" + os.Args[3]
			if inGame {
				t = 40
				gt := GameTurn{
					action:   MOVE,
					move:     message,
					gameUUID: gameUUID,
					client:   os.Args[3],
				}
				sendTLV(conn, t, gt.Encode(clientKey, encryptionKey))
			}
			if strings.Contains(message, " ") {
				splittedMessage := strings.Split(message, " ")
				switch splittedMessage[0] {
				case "play":
					gr := GameRequest{
						action:   PLAY,
						gameType: SOLO,
						client:   os.Args[3],
					}
					sendTLV(conn, 30, gr.Encode(clientKey))

				default:
					sendTLV(conn, t, []byte(message+"|"+messageSigned))
				}
			} else {
				//sendTLV(conn, t, []byte(message+"|"+messageSigned))
			}
		} else {
			sendTLV(conn, t, []byte(message+"|"+messageSigned))
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

func handleTLV(tag byte, data []byte, serverKey *string, encryptionKey *string, gameUUID *string, inGame *bool) {
	switch tag {
	case 100: // Auth
		*serverKey = string(data)
		fmt.Println("Écrivez le chiffre de l'action voulu: \n1 - Play\n2 - Join\n3 - Exit")

	case 101: // Action
		splitedMessage := strings.Split(string(data), "|")
		if signMessage(*serverKey, splitedMessage[0]) == splitedMessage[1] {
			fmt.Println("> " + splitedMessage[0])
		} else {
			fmt.Println("bad packet")
		}
	case 130:
		signature := ""
		accumulatedData := ""
		board := ""

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
			default:

			}
		})

		if signMessage(*serverKey, accumulatedData) == signature {
			fmt.Println(board)
			*inGame = true
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
				fmt.Println("L'adversaire à joué: " + serverMove)
				fmt.Println(board)
				fmt.Print("> Entrez votre coup: ")
			}

		} else {
			fmt.Println("Bad packet")
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
