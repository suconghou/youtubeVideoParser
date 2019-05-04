package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/suconghou/youtubeVideoParser"
)

func main1() {
	var id = os.Args[1]
	if id == "" {
		id = "awa2Nm8B5_I"
	}
	var (
		parser, err = youtubeVideoParser.NewParser(id)
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	info, err := parser.Parse()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(info)
	for i, v := range info.Streams {
		fmt.Println(i, v)
	}
}

func main() {
	var port = 9977
	http.HandleFunc("/video", routeMatch)
	http.ListenAndServe(fmt.Sprintf("%s:%d", os.Getenv("HOST"), port), nil)
}

func routeMatch(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	var query = r.URL.Query()
	var id = query.Get("id")
	var (
		parser, err = youtubeVideoParser.NewParser(id)
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	info, err := parser.Parse()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(info)
	for i, v := range info.Streams {
		fmt.Println(i, v)
	}
	bs, err := json.Marshal(info)
	if err != nil {
		fmt.Println(err)
		return
	}
	w.Write(bs)
}
