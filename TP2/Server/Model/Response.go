package Model

import (
	utils "TP2/Utils"
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/notnil/chess"
)

type GameResponse struct {
	GameUUID      string
	GameFEN       string
	PlayerList    string
	EncryptionKey string
	Color         byte
}

func (gr GameResponse) Encode(serverKey string) []byte {
	fen, err := chess.FEN(gr.GameFEN)
	if err != nil {
		fmt.Println(err.Error())
	}
	accumulatedData := ""
	gameUUIDByte := utils.BuildSubTLV(131, []byte(gr.GameUUID))
	keyByte := utils.BuildSubTLV(4, []byte(gr.EncryptionKey))

	if gr.GameFEN == "" {
		playerListByte := utils.BuildSubTLV(133, []byte(gr.PlayerList))
		accumulatedData += gr.GameUUID + gr.PlayerList + gr.EncryptionKey
		binary.Write(&gameUUIDByte, binary.BigEndian, playerListByte.Bytes())
	} else {
		gameFENByte := utils.BuildSubTLV(132, []byte(chess.NewGame(fen).Position().Board().Draw()))
		accumulatedData += gr.GameUUID + chess.NewGame(fen).Position().Board().Draw() + gr.EncryptionKey
		binary.Write(&gameUUIDByte, binary.BigEndian, gameFENByte.Bytes())
	}
	binary.Write(&gameUUIDByte, binary.BigEndian, keyByte.Bytes())
	if gr.Color != 0 {
		colorByte := utils.BuildSubTLV(134, []byte{gr.Color})
		accumulatedData += string(gr.Color)
		binary.Write(&gameUUIDByte, binary.BigEndian, colorByte.Bytes())
		turnColor := UNDEFINED
		switch chess.NewGame(fen).Position().Turn().String() {
		case "b":
			turnColor = BLACK
		case "w":
			turnColor = WHITE
		default:
			turnColor = UNDEFINED
		}
		turnByte := utils.BuildSubTLV(135, []byte{turnColor})
		accumulatedData += string(turnColor)
		binary.Write(&gameUUIDByte, binary.BigEndian, turnByte.Bytes())
	}
	signatureByte := utils.BuildSubTLV(3, []byte(utils.SignMessage(serverKey, accumulatedData)))
	binary.Write(&gameUUIDByte, binary.BigEndian, signatureByte.Bytes())
	return gameUUIDByte.Bytes()
}

type GameActionResponse struct {
	Action       byte
	GameUUID     string
	GameFEN      string
	MoveResponse string
	ServerMove   string
	BestMove     string
	Outcome      string
	Turn         byte
	Err          string
}

func (gar GameActionResponse) Encode(serverKey string, encryptionKey string) []byte {
	response := new(bytes.Buffer)
	fen, err := chess.FEN(gar.GameFEN)
	if err != nil {
		fmt.Println(err.Error())
	}
	board := chess.NewGame(fen).Position().Board().Draw()
	if gar.Action == OPPONENT_MOVE_RESPONSE {
		board = chess.NewGame(fen).Position().Board().Draw()
	}

	action := utils.BuildSubTLV(141, []byte{gar.Action})
	gameUUIDByte := utils.BuildSubTLV(131, []byte(gar.GameUUID))
	gameBoardByte := utils.BuildSubTLV(132, []byte(board))
	signatureData := string(gar.Action) + gar.GameUUID + board
	accumulatedData := action.String() + gameUUIDByte.String() + gameBoardByte.String()

	switch gar.Action {
	case MOVE_RESPONSE, OPPONENT_MOVE_RESPONSE:
		moveResponseByte := utils.BuildSubTLV(142, []byte(gar.MoveResponse)) // à modifié
		accumulatedData += moveResponseByte.String()
		signatureData += gar.MoveResponse
		if gar.ServerMove != "" {
			serverMoveByte := utils.BuildSubTLV(143, []byte(gar.ServerMove))
			accumulatedData += serverMoveByte.String()
			signatureData += gar.ServerMove
		} else {
			if gar.Turn != UNDEFINED {
				turnByte := utils.BuildSubTLV(134, []byte{gar.Turn})
				accumulatedData += turnByte.String()
				signatureData += string(gar.Turn)
			}
		}

	case GAME_OUTCOME:
		moveResponseByte := utils.BuildSubTLV(142, []byte(gar.MoveResponse))
		if gar.ServerMove != "" {
			serverMoveByte := utils.BuildSubTLV(142, []byte(gar.ServerMove))
			accumulatedData += serverMoveByte.String()
			signatureData += gar.ServerMove
		}
		outcomeByte := utils.BuildSubTLV(144, []byte(gar.Outcome))
		accumulatedData += outcomeByte.String() + moveResponseByte.String()
		signatureData += gar.Outcome + gar.MoveResponse
	case ERROR:
		errorByte := utils.BuildSubTLV(199, []byte(gar.Err))
		if gar.BestMove != "" {
			bestMoveByte := utils.BuildSubTLV(145, []byte(gar.BestMove))
			accumulatedData += bestMoveByte.String()
			signatureData += gar.BestMove
		}
		accumulatedData += errorByte.String()
		signatureData += gar.Err
	}
	signature := utils.SignMessage(serverKey, signatureData)
	signatureByte := utils.BuildSubTLV(3, []byte(signature))
	accumulatedData += signatureByte.String()
	accumulatedData, err = utils.Encrypt(accumulatedData, encryptionKey)
	if err != nil {
		fmt.Println(err.Error())
	}
	binary.Write(response, binary.BigEndian, []byte(accumulatedData))
	return response.Bytes()
}

type GameListResponse struct {
	List  string
	Error string
}

func (glr GameListResponse) Encode(serveyKey string) []byte {
	response := new(bytes.Buffer)
	signatureData := glr.List + glr.Error
	listByte := utils.BuildSubTLV(151, []byte(glr.List))
	errorByte := utils.BuildSubTLV(152, []byte(glr.Error))
	accumulatedData := listByte.String() + errorByte.String()

	signature := utils.SignMessage(serveyKey, signatureData)
	signatureByte := utils.BuildSubTLV(3, []byte(signature))
	accumulatedData += signatureByte.String()
	binary.Write(response, binary.BigEndian, []byte(accumulatedData))
	return response.Bytes()
}
