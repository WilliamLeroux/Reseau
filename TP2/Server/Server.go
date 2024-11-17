package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

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

	for {
		// Read from the connection untill a new line is send
		data, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		// Print the data read from the connection to the terminal
		fmt.Print("> ", string(data))

		// Write back the same message to the client
		conn.Write([]byte("Hello " + data + "\n"))
	}
}

func UDPServer(wg *sync.WaitGroup) {
	defer wg.Done()
	//isConnected := false
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

	for {

		//var buf [512]byte
		buffer := make([]byte, 1024)
		_, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("> ", string(buffer))

		handleTLVUDP(conn, addr, buffer)

		// Write back the message over UPD
		/*
			if !isConnected {
				conn.WriteToUDP([]byte("Hello "+string(buf[0:])+"\n"), addr)
				isConnected = true
			} else {
				conn.WriteToUDP([]byte(string(buf[0:])), addr)
			}
		*/
	}
}

func handleTLVUDP(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	fmt.Println("TLV")
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
	case 1: // Authentification
		//client := string(value)
		fmt.Println("Tag 1")
		secret := auth() // Ajouter le nom du client

		secretResponse := buildTLV(2, []byte(secret))
		_, err := conn.WriteToUDP(secretResponse, addr)
		if err != nil {
			fmt.Println(err.Error())
		}
		// Vérification utilisateur existe déjà
		// Si il n'existe pas, ajout à la base de donnée et
	case 2:
		fmt.Println(value)
	default:
		fmt.Println("Tag inconnu")
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

func auth() string {
	// Vérification si le client existe ou non, si oui envoie son uuid sinon en créer un nouveau et l'insérer dans la bd
	secret := generateUUID()
	return secret
}

func generateUUID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Int31(), rand.Int31n(0xFFFF), rand.Int31n(0xFFFF),
		rand.Int31n(0xFFFF), rand.Int63n(0xFFFFFFFFFFFF))
}
