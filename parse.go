package youtubeVideoParser

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	_ "os"
	"regexp"
	"strings"
)

var debug bool = true

var ytplayer_config_regexp *regexp.Regexp = regexp.MustCompile(`ytplayer.config\s*=\s*([^\n]+?});`)

var youtube_video_host string = "https://www.youtube.com/get_video_info?video_id=%s"
var video_page_host string = "https://www.youtube.com/watch?v=%s"
var html5player_host string = "https://www.youtube.com%s"

type ytplayerConfig struct {
	Args struct {
		Url_encoded_fmt_stream_map string
		Author                     string
		Length_seconds             string
		Title                      string
		Keywords                   string
		View_count                 string
		Thumbnail_url              string
		Video_id                   string
		Caption_tracks             string
		Dashmpd                    string
		Livestream                 string
		Live_playback              string
		Hlsvp                      string
		Adaptive_fmts              string
	}
	Assets struct {
		Css string
		Js  string
	}
}

func Parse(id string) (videoInfo, error) {
	var info videoInfo
	var u string = fmt.Sprintf(video_page_host, id)
	res, err := HttpGet(fmt.Sprintf(youtube_video_host, id))
	if err != nil {
		return info, err
	}
	values, err := url.ParseQuery(string(res))
	if err != nil {
		return info, err
	}
	if status, ok := values["status"]; !ok {
		log("no status")
		return info, fmt.Errorf("error no status found in get_video_info")
	} else if status[0] == "ok" {
		video_page, err := HttpGet(u)
		if err != nil {
			return info, err
		}
		var parsePage bool
		if _, ok := values["use_cipher_signature"]; !ok {
			parsePage = false
		} else if v, ok := values["use_cipher_signature"]; ok && v[0] == "False" {
			parsePage = false
		} else {
			parsePage = true
		}
		if parsePage {

		}
		ytplayer_config_matches := ytplayer_config_regexp.FindStringSubmatch(string(video_page))
		info.Id = values["video_id"][0]
		info.Title = values["title"][0]
		info.Duration = values["length_seconds"][0]
		info.Keywords = values["keywords"][0]
		info.Author = values["author"][0]
		if len(ytplayer_config_matches) > 1 {
			cfgInfo := &ytplayerConfig{}
			err := json.Unmarshal([]byte(ytplayer_config_matches[1]), &cfgInfo)
			if err != nil {
				return info, err
			}
			var html5playerUrl string = fmt.Sprintf(html5player_host, cfgInfo.Assets.Js)
			var stream_list []string = strings.Split(cfgInfo.Args.Url_encoded_fmt_stream_map, ",")
			streams, err := parseStream(stream_list)
			if err != nil {
				return info, err
			}
			info.Stream = streams
			if html5playerUrl != "" {

			}
			return info, nil
		} else {
			log("ytplayer_config not match in page source")
			stream_list := strings.Split(values["url_encoded_fmt_stream_map"][0], ",")
			streams, err := parseStream(stream_list)
			info.Stream = streams
			if err != nil {
				return info, err
			}
			return info, nil
		}
	} else if status[0] == "fail" {
		log("get get_video_info failed")
		if values["errorcode"][0] == "150" {
			video_page, err := HttpGet(u)
			if err != nil {
				return info, err
			}
			ytplayer_config_matches := ytplayer_config_regexp.FindStringSubmatch(string(video_page))
			if len(ytplayer_config_matches) > 1 {
				cfgInfo := &ytplayerConfig{}
				err := json.Unmarshal([]byte(ytplayer_config_matches[1]), &cfgInfo)
				if err != nil {
					return info, err
				}
				if cfgInfo.Args.Title != "" {
					info.Title = cfgInfo.Args.Title
					info.Duration = cfgInfo.Args.Length_seconds
					info.Id = cfgInfo.Args.Video_id
					info.Keywords = cfgInfo.Args.Keywords
					info.Author = cfgInfo.Args.Author
					var stream_list []string = strings.Split(cfgInfo.Args.Url_encoded_fmt_stream_map, ",")
					streams, err := parseStream(stream_list)
					if err != nil {
						return info, err
					}
					info.Stream = streams
					return info, nil
				} else {
					return info, fmt.Errorf("[Error] The uploader has not made this video available in your country.")
				}
			} else {
				return info, fmt.Errorf("[Failed] ")
			}
		} else if values["errorcode"][0] == "100" {
			return info, fmt.Errorf("[Failed] This video does not exist.")
		} else {
			return info, fmt.Errorf("[Failed] %s", values["reason"][0])
		}
	} else {
		log("unknow status")
		return info, fmt.Errorf("[Failed] unknow status")
	}
	return info, err
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

func parseStream(stream_list []string) (map[string]streamItem, error) {
	res := map[string]streamItem{}
	for _, item := range stream_list {
		metadata, _ := url.ParseQuery(item)
		stream_itag := metadata["itag"][0]
		var sig, s, mime string
		if v, ok := metadata["sig"]; ok {
			sig = v[0]
		}
		if v, ok := metadata["s"]; ok {
			s = v[0]
		}
		if v, ok := metadata["type"]; ok {
			arr := strings.Split(v[0], ";")
			mime = arr[0]
		}
		res[stream_itag] = streamItem{
			Itag:      stream_itag,
			Url:       metadata["url"][0],
			Sig:       sig,
			S:         s,
			Quality:   metadata["quality"][0],
			Type:      metadata["type"][0],
			Mime:      mime,
			Container: mime_to_container(mime),
		}
	}
	return res, nil
}

func mime_to_container(mime string) string {
	if v, ok := mimeMap[mime]; ok {
		return v
	}
	return strings.Split(mime, "/")[1]
}
