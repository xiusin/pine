package main

import (
	"fmt"
	"net/http"
)

func main() {
	client := http.Client{}
	req, err := http.NewRequest("GET", "https://d3ljqgx1ze2ak9.cloudfront.net/online/videos/2018-08-22/26/35b7cf6f824f5d7.96518277_fps25.mp4", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", "video/mp4")
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Header)
}
