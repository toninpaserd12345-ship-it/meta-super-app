package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	url := "https://graph.facebook.com/v20.0/4/picture?redirect=true&type=large"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "image/*")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("ContentType:", resp.Header.Get("Content-Type"))
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Body length:", len(body))
}
