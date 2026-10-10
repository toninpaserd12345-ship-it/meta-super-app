package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	url := "https://graph.facebook.com/v20.0/4?fields=id,name,picture.type(large)"
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Body:", string(body))
}
