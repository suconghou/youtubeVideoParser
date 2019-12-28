package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/suconghou/youtubevideoparser"
)

func main() {
	if len(os.Args) > 1 {
		info, err := getInfo(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(info.Title)
		for i, v := range info.Streams {
			fmt.Println(i, v)
		}
		return
	}
	http.HandleFunc("/video", routeMatch)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", os.Getenv("HOST"), 9977), nil))
}

func routeMatch(w http.ResponseWriter, r *http.Request) {
	var (
		query = r.URL.Query()
		id    = query.Get("id")
	)
	if id == "" {
		http.Error(w, "error id", http.StatusNotFound)
		return
	}
	info, err := getInfo(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bs, err := json.Marshal(info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bs)
}

func getInfo(id string) (*youtubevideoparser.VideoInfo, error) {
	var (
		parser, err = youtubevideoparser.NewParser(id)
	)
	if err != nil {
		return nil, err
	}
	info, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return info, err
}
