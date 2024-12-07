package utils

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

func Mapkey(m map[string]*net.UDPAddr, value *net.UDPAddr) (key string, ok bool) {
	for k, v := range m {
		if v.String() == value.String() {
			key = k
			ok = true
			return
		}
	}
	return
}

func GenerateUUID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Int31(), rand.Int31n(0xFFFF), rand.Int31n(0xFFFF),
		rand.Int31n(0xFFFF), rand.Int63n(0xFFFFFFFFFFFF))
}
