package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"os"
)

func main() {
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

	sendTLV(conn, 1, []byte(os.Args[3]))
	/*
		_, err = conn.Write([]byte(os.Args[3] + "\n"))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}*/
	messageReader := bufio.NewReader(os.Stdin)

	buffer := make([]byte, 1024)
	// Read from the connection untill a new line is send
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("apres reader")
		// Print the data read from the connection to the terminal
		tag, value := parseTLV(buffer[:n])
		fmt.Print("> tag:" + string(tag) + "\n> value: " + string(value))

		message, err := messageReader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}
		sendTLV(conn, 2, []byte(message))
		//fmt.Fprintf(conn, message+"\n")
	}
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

func signMessage(secretKey, message string) string {
	data := secretKey + message
	hash := sha256.New()
	hash.Write([]byte(data))
	signature := hash.Sum(nil)
	return fmt.Sprintf("%x", signature)
}
