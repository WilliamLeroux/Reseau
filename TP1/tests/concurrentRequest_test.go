package tests

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

func sendRequest(wg *sync.WaitGroup, url string) {
	defer wg.Done()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)
}

func TestCreateDeck(t *testing.T) {
	var wg sync.WaitGroup
	url := "http://localhost:8080/deck/new/1/false"
	concurrentUsers := 100

	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go sendRequest(&wg, url)
	}

	wg.Wait()
	t.Log("All requests completed")
}
