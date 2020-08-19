package youtubevideoparser

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/suconghou/youtubevideoparser/request"
	"github.com/tidwall/gjson"
)

const (
	baseURL       = "https://www.youtube.com"
	videoPageHost = baseURL + "/watch?v=%s"
	videoInfoHost = baseURL + "/get_video_info?video_id=%s"
)

var ytplayerConfigRegexp = regexp.MustCompile(`;ytplayer\.config\s*=\s*({.+?});ytplayer`)

// Parser return instance
type Parser struct {
	ID     string
	JsPath string
	Player gjson.Result
}

// VideoInfo contains video info
type VideoInfo struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Duration string                 `json:"duration"`
	Author   string                 `json:"author"`
	Streams  map[string]*StreamItem `json:"streams"`
}

// NewParser create Parser instance
func NewParser(id string) (*Parser, error) {
	var (
		videoPageURL = fmt.Sprintf(videoPageHost, id)
		cachekey     = "jsPath"
	)
	videoPageData, err := request.GetURLData(videoPageURL, false)
	if err != nil {
		return nil, err
	}
	var (
		jsPath string
		player gjson.Result
	)
	if arr := ytplayerConfigRegexp.FindSubmatch(videoPageData); len(arr) >= 2 {
		res := gjson.ParseBytes(arr[1])
		jsPath = res.Get("assets.js").String()
		player = gjson.Parse(res.Get("args.player_response").String())
		if jsPath != "" {
			request.Set(cachekey, []byte(jsPath))
		}
	}
	if jsPath == "" {
		var (
			videoInfoURL = fmt.Sprintf(videoInfoHost, id)
		)
		videoInfoData, err := request.GetURLData(videoInfoURL, false)
		if err != nil {
			return nil, err
		}
		values, err := url.ParseQuery(string(videoInfoData))
		if err != nil {
			return nil, err
		}
		status := values.Get("status")
		if status != "ok" {
			return nil, fmt.Errorf("%s %s:%s", status, values.Get("errorcode"), values.Get("reason"))
		}
		player = gjson.Parse(values.Get("player_response"))
		jsPath = string(request.Get(cachekey))
	}
	return &Parser{
		id,
		jsPath,
		player,
	}, nil
}

// Parse parse video info
func (p *Parser) Parse() (*VideoInfo, error) {
	var (
		v    = p.Player.Get("videoDetails")
		info = &VideoInfo{
			ID:       p.ID,
			Title:    v.Get("title").String(),
			Duration: v.Get("lengthSeconds").String(),
			Author:   v.Get("author").String(),
			Streams:  make(map[string]*StreamItem),
		}
		s   = p.Player.Get("streamingData")
		err error
	)
	var loop = func(key gjson.Result, value gjson.Result) bool {
		var (
			url           string
			itag          = value.Get("itag").String()
			streamType    = value.Get("mimeType").String()
			quality       = value.Get("qualityLabel").String()
			contentLength = value.Get("contentLength").String()
		)
		if quality == "" {
			quality = value.Get("quality").String()
		}
		if value.Get("url").Exists() {
			url = value.Get("url").String()
		} else if value.Get("cipher").Exists() {
			url, err = buildURL(value.Get("cipher").String(), p.JsPath)
		} else if value.Get("signatureCipher").Exists() {
			url, err = buildURL(value.Get("signatureCipher").String(), p.JsPath)
		}
		info.Streams[itag] = &StreamItem{
			quality,
			streamType,
			url,
			itag,
			contentLength,
			&rangeItem{
				value.Get("initRange.start").String(),
				value.Get("initRange.end").String(),
			},
			&rangeItem{
				value.Get("indexRange.start").String(),
				value.Get("indexRange.end").String(),
			},
		}
		return true
	}
	s.Get("formats").ForEach(loop)
	s.Get("adaptiveFormats").ForEach(loop)
	return info, err
}

func buildURL(cipher string, jsPath string) (string, error) {
	var (
		stream, err = url.ParseQuery(cipher)
	)
	if err != nil {
		return "", err
	}
	if jsPath == "" {
		return "", fmt.Errorf("jsPath not found")
	}
	bodystr, err := request.GetURLData(baseURL+jsPath, true)
	if err != nil {
		return "", err
	}
	return getDownloadURL(stream, string(bodystr))
}
