package Model

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/notnil/chess"
)

type GameResponse struct {
	gameUUID      string
	gameFEN       string
	playerList    string
	encryptionKey string
	yourTurn      bool
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
		fmt.Println("fen vide")
		playerListByte := BuildSubTLV(133, []byte(gr.playerList))
		accumulatedData += gr.gameUUID + gr.playerList + gr.encryptionKey
		binary.Write(&gameUUIDByte, binary.BigEndian, playerListByte.Bytes())
	} else {
		gameFENByte := BuildSubTLV(132, []byte(chess.NewGame(fen).Position().Board().Draw()))
		accumulatedData += gr.gameUUID + chess.NewGame(fen).Position().Board().Draw() + gr.encryptionKey
		binary.Write(&gameUUIDByte, binary.BigEndian, gameFENByte.Bytes())
		/*
			if &gr.yourTurn != nil {
				if gr.yourTurn {
					turnByte := BuildSubTLV(134, []byte{0})
					accumulatedData += string(0)
					binary.Write(&gameUUIDByte, binary.BigEndian, turnByte.Bytes())
				} else {
					turnByte := BuildSubTLV(134, []byte{1})
					accumulatedData += string(1)
					binary.Write(&gameUUIDByte, binary.BigEndian, turnByte.Bytes())
				}
			}*/
	}
	signatureByte := BuildSubTLV(3, []byte(signMessage(serverKey, accumulatedData)))
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
	case MOVE_RESPONSE:
		moveResponseByte := BuildSubTLV(142, []byte(gar.moveResponse)) // à modifié
		accumulatedData += moveResponseByte.String()
		signatureData += gar.moveResponse
		if gar.serverMove != "" {
			serverMoveByte := BuildSubTLV(143, []byte(gar.serverMove))
			accumulatedData += serverMoveByte.String()
			signatureData += gar.serverMove
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
	signature := signMessage(serverKey, signatureData)
	signatureByte := BuildSubTLV(3, []byte(signature))
	accumulatedData += signatureByte.String()
	//binary.Write(&value, binary.BigEndian, signatureByte.Bytes())
	accumulatedData, err = Encrypt(accumulatedData, encryptionKey)
	if err != nil {
		fmt.Println(err.Error())
	}
	binary.Write(response, binary.BigEndian, []byte(accumulatedData))
	return response.Bytes()
}

type GameInviteResponse struct {
	gameUUID string
}

func (gir GameInviteResponse) Encode(serverKey string) []byte {
	signature := BuildSubTLV(3, []byte(signMessage(serverKey, gir.gameUUID)))
	request := new(bytes.Buffer)
	gameUUIDByte := BuildSubTLV(131, []byte(gir.gameUUID))
	binary.Write(request, binary.BigEndian, gameUUIDByte.Bytes())
	binary.Write(request, binary.BigEndian, signature.Bytes())
	return request.Bytes()
}
