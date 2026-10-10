package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	var response struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Category    string `json:"category"`
			AccessToken string `json:"access_token"`
			Picture     struct {
				Data struct {
					URL string `json:"url"`
				} `json:"data"`
			} `json:"picture"`
		} `json:"data"`
	}

	payload := `{"data": [{"id": "1", "name": "Test", "category": "Test", "access_token": "token", "picture": {"data": {"url": "https://example.com/pic.jpg"}}}]}`
	json.Unmarshal([]byte(payload), &response)
	fmt.Println("URL:", response.Data[0].Picture.Data.URL)
}
