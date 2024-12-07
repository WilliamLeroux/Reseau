package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

func BuildTLV(tag byte, value []byte) []byte {
	length := uint16(len(value))
	buffer := make([]byte, 3+length)
	buffer[0] = tag
	binary.BigEndian.PutUint16(buffer[1:3], length)
	copy(buffer[3:], value)
	return buffer
}

func BuildSubTLV(tag byte, value []byte) bytes.Buffer {
	length := uint16(len(value))
	buffer := new(bytes.Buffer)
	buffer.WriteByte(tag)
	binary.Write(buffer, binary.BigEndian, length)
	buffer.Write(value)
	return *buffer
}

func ParseSubTLV(data []byte, handleSubTLV func(byte, []byte)) {
	offset := 0
	for offset < len(data) {
		subTag := data[offset]
		subLength := binary.BigEndian.Uint16(data[offset+1 : offset+3])
		subValue := data[offset+3 : offset+3+int(subLength)]
		handleSubTLV(subTag, subValue)
		offset += 3 + int(subLength)
	}
}

func SignMessage(secretKey, message string) string {
	data := secretKey + message
	hash := sha256.New()
	hash.Write([]byte(data))
	signature := hash.Sum(nil)
	return fmt.Sprintf("%x", signature)
}
