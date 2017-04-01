package youtubeVideoParser

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var debug bool = false

var youtube_video_host string = "http://www.youtube.com/get_video_info?video_id="

func Parse(id string) (videoInfo, error) {
	var info videoInfo
	var u string = fmt.Sprintf("%s%s%s", youtube_video_host, id, "&asv=3&el=detailpage&hl=en_US")
	res, err := HttpGet(u)
	if err != nil {
		return info, err
	}
	info, err = getVideoInfo(string(res))
	return info, err
}

func getVideoInfo(res string) (videoInfo, error) {
	var info videoInfo
	values, err := url.ParseQuery(res)
	err = ensureFields(values, []string{"status", "url_encoded_fmt_stream_map", "title", "author", "length_seconds", "keywords", "video_id"})
	if err != nil {
		return info, err
	}
	status := values["status"]
	if status[0] == "fail" {
		reason, ok := values["reason"]
		if ok {
			return info, fmt.Errorf("'fail' response status found in the server's answer, reason: '%s'", reason[0])
		} else {
			return info, fmt.Errorf("'fail' response status found in the server's answer, no reason given")
		}
	}
	if status[0] != "ok" {
		return info, fmt.Errorf("non-success response status found in the server's answer (status: '%s')", status)
	}
	info.Id = values["video_id"][0]
	info.Title = values["title"][0]
	info.Duration = values["length_seconds"][0]
	info.Keywords = values["keywords"][0]
	info.Author = values["author"][0]
	streams, err := decodeVideoInfo(values)
	if err != nil {
		return info, err
	}
	info.Stream = streams
	return info, nil
}

func decodeVideoInfo(values url.Values) (streamList, error) {
	var streams streamList
	stream_map := values["url_encoded_fmt_stream_map"]
	streams_list := strings.Split(stream_map[0], ",")
	for stream_pos, stream_raw := range streams_list {
		stream_qry, err := url.ParseQuery(stream_raw)
		if err != nil {
			log(fmt.Sprintf("An error occured while decoding one of the video's stream's information: stream %d: %s\n", stream_pos, err))
			continue
		}
		err = ensureFields(stream_qry, []string{"quality", "type", "url"})
		if err != nil {
			log(fmt.Sprintf("Missing fields in one of the video's stream's information: stream %d: %s\n", stream_pos, err))
			continue
		}
		stream := stream{
			"quality": tQuality(stream_qry["quality"][0]),
			"type":    tFormat(stream_qry["type"][0]),
			"url":     stream_qry["url"][0],
		}
		var streamsig string
		if sig, exists := stream_qry["sig"]; exists { // old one
			streamsig = sig[0]
		}
		if sig, exists := stream_qry["s"]; exists { // now they use this
			streamsig = sig[0]
		}
		stream["url"] = tUrl(stream["url"], streamsig)
		streams = append(streams, stream)

		log("Stream found: quality '%s', format '%s'", stream["quality"], stream["type"])
	}
	log("Successfully decoded %d streams", len(streams))
	return streams, nil
}

func HttpGet(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	str, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return str, err
	}
	return str, nil
}

func ensureFields(source url.Values, fields []string) error {
	for _, field := range fields {
		if _, exists := source[field]; !exists {
			return fmt.Errorf("Field '%s' is missing in url.Values source", field)
		}
	}
	return nil
}

func log(format string, params ...interface{}) {
	if debug {
		fmt.Printf(format+"\n", params...)
	}
}
