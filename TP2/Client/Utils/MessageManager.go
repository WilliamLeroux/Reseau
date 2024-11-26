package utils

import (
	"bufio"
	"fmt"
	"strings"
)

/**
 * @brief Li la console
 * @param messageReader le reader
 * @return string sans saut de ligne
 */
func ReadConsole(messageReader *bufio.Reader) string {
	fmt.Print("> ")
	message, err := messageReader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return strings.ReplaceAll(message, "\n", "")
}
