package youtubevideoparser

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/suconghou/youtubevideoparser/request"
	"github.com/tidwall/gjson"
)

const (
	baseURL       = "https://www.youtube.com"
	videoPageHost = baseURL + "/watch?v=%s"
)

var (
	jsPathRegexp     = regexp.MustCompile(`"jsUrl":"(/s/player.*?base.js)"`)
	initPlayerRegexp = regexp.MustCompile(`ytInitialPlayerResponse\s+=\s+(.*}+);\s*var`)
	jsPath           string
)

// Parser return instance
type Parser struct {
	ID     string
	Player gjson.Result
	client http.Client
}

// VideoInfo contains video info
type VideoInfo struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Duration string                 `json:"duration"`
	Author   string                 `json:"author"`
	Captions []*Caption             `json:"captions,omitempty"`
	Streams  map[string]*StreamItem `json:"streams"`
}

// NewParser create Parser instance
func NewParser(id string, client http.Client) (*Parser, error) {
	var (
		videoPageURL = fmt.Sprintf(videoPageHost, id)
		player       gjson.Result
		ok           = false
	)
	if bs, err := request.CachePost(id, client); err == nil {
		player = gjson.ParseBytes(bs)
		if player.Get("playabilityStatus.status").String() == "OK" && player.Get("videoDetails").Exists() && player.Get("streamingData").Exists() {
			ok = true
		}
	}
	if !ok {
		videoPageData, err := request.CacheGet(videoPageURL, client)
		if err != nil {
			return nil, err
		}
		if arr := jsPathRegexp.FindSubmatch(videoPageData); len(arr) >= 2 {
			jsPath = string(arr[1])
		}
		if arr := initPlayerRegexp.FindSubmatch(videoPageData); len(arr) >= 2 {
			player = gjson.ParseBytes(arr[1])
			status := player.Get("playabilityStatus.status").String()
			if status != "OK" || !player.Get("videoDetails").Exists() || !player.Get("streamingData").Exists() {
				ps := player.Get("playabilityStatus")
				reason := ps.Get("reason").String()
				return nil, fmt.Errorf("%s %s %s", id, status, reason)
			}
		} else {
			return nil, fmt.Errorf("%s failed to get ytInitialPlayerResponse", id)
		}
	}
	return &Parser{
		id,
		player,
		client,
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
			Captions: parseCaptions(p.Player),
			Streams:  make(map[string]*StreamItem),
		}
		s          = p.Player.Get("streamingData")
		err        error
		cipherBody string
	)
	var buildURL = func(cipher string) (string, error) {
		stream, err := url.ParseQuery(cipher)
		if err != nil {
			return "", err
		}
		if cipherBody == "" {
			if jsPath == "" {
				return "", fmt.Errorf("jsPath not found")
			}
			bs, err := request.CacheGetLong(baseURL+jsPath, p.client)
			if err != nil {
				return "", err
			}
			cipherBody = string(bs)
		}
		return getDownloadURL(stream, cipherBody)
	}
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
			url, err = buildURL(value.Get("cipher").String())
		} else if value.Get("signatureCipher").Exists() {
			url, err = buildURL(value.Get("signatureCipher").String())
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
		if err != nil {
			return false
		}
		return true
	}
	s.Get("formats").ForEach(loop)
	s.Get("adaptiveFormats").ForEach(loop)
	return info, err
}

func parseCaptions(player gjson.Result) []*Caption {
	var captions = []*Caption{}
	var loop = func(key gjson.Result, value gjson.Result) bool {
		var l = value.Get("name.simpleText").String()
		if l == "" {
			l = value.Get("name.runs[0].text").String()
		}
		captions = append(captions, &Caption{
			URL:          value.Get("baseUrl").String(),
			Language:     value.Get("name.simpleText").String(),
			LanguageCode: value.Get("languageCode").String(),
		})
		return true
	}
	player.Get("captions.playerCaptionsTracklistRenderer.captionTracks").ForEach(loop)
	return captions
}
