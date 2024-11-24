package main

import "net"

func mapkey(m map[string]*net.UDPAddr, value *net.UDPAddr) (key string, ok bool) {
	for k, v := range m {
		if v.String() == value.String() {
			key = k
			ok = true
			return
		}
	}
	return
}

func mapKeyString(m map[string]string, value string) (key string, ok bool) {
	for k, v := range m {
		if v == value {
			key = k
			ok = true
			return
		}
	}
	return
}
