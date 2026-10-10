package main

import (
	"fmt"
	"net/http"
)

func main() {
	// A public Facebook page ID (e.g. 4 for Mark Zuckerberg, or a random page)
	url := "https://graph.facebook.com/4/picture?type=large"
	req, _ := http.NewRequest("GET", url, nil)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // don't follow redirect
		},
	}
	resp, _ := client.Do(req)
	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Location:", resp.Header.Get("Location"))
}
