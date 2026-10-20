package youtubevideoparser

import (
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/suconghou/youtubevideoparser/request"
	"github.com/tidwall/gjson"
)

const (
	baseURL       = "https://www.youtube.com"
	videoPageHost = baseURL + "/watch?v=%s"
)

var (
	initPlayerRegexp = regexp.MustCompile(`ytInitialPlayerResponse\s+=\s+(.*}+);\s*var`)
	caches           = sync.Map{}
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

type cacheInfo struct {
	data *VideoInfo
	time time.Time
}

// 包含一层缓存包装
func Parse(id string, client http.Client) (*VideoInfo, error) {
	var now = time.Now()
	caches.Range(func(key, value any) bool {
		var v = value.(*cacheInfo)
		if now.Sub(v.time) > time.Hour {
			caches.Delete(key)
		}
		return true
	})
	if v, ok := caches.Load(id); ok {
		return v.(*cacheInfo).data, nil
	}
	parser, err := NewParser(id, client)
	if err != nil {
		return nil, err
	}
	info, err := parser.Parse()
	if err == nil {
		caches.Store(id, &cacheInfo{
			info,
			now,
		})
	}
	return info, err
}

// NewParser仅包含http请求缓存,若要完全的缓存请使用Parse()
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
		// API 已返回但不可播：LOGIN_REQUIRED 是 IP 级风控，页面解析同样失败，无需 fallback
		if status := player.Get("playabilityStatus.status").String(); status != "" {
			ps := player.Get("playabilityStatus")
			reason := ps.Get("reason").String()
			if reason == "" {
				reason = status
			}
			return nil, fmt.Errorf("%s %s %s", id, status, reason)
		}
		videoPageData, err := request.CacheGet(videoPageURL, client)
		if err != nil {
			return nil, err
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
		s   = p.Player.Get("streamingData")
		err error
	)
	var buildURL = func(cipher string) (string, error) {

		return "", fmt.Errorf("not implemented")
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
		return err == nil
	}
	s.Get("formats").ForEach(loop)
	s.Get("adaptiveFormats").ForEach(loop)
	return info, err
}

func parseCaptions(player gjson.Result) []*Caption {
	var captions = []*Caption{}
	var loop = func(key gjson.Result, value gjson.Result) bool {
		captions = append(captions, &Caption{
			URL:      value.Get("baseUrl").String(),
			Language: value.Get("languageCode").String(),
		})
		return true
	}
	player.Get("captions.playerCaptionsTracklistRenderer.captionTracks").ForEach(loop)
	return captions
}
