package main

import (
	"encoding/json"
	"fmt"
	"github.com/meta-super-app/backend/internal/domain"
)

func main() {
	page := domain.MetaPage{
		ID:         "123",
		Name:       "Test",
		PictureURL: "https://example.com/pic.jpg",
	}
	b, _ := json.Marshal(page)
	fmt.Println(string(b))
}
