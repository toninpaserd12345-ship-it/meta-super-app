package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	url := "https://graph.facebook.com/v20.0/4/picture?redirect=0&type=large"
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Body:", string(body))
}
