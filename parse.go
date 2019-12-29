package youtubevideoparser

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/suconghou/youtubevideoparser/request"
	"github.com/tidwall/gjson"
)

const (
	baseURL       = "https://www.youtube.com"
	videoPageHost = baseURL + "/watch?v=%s"
)

var (
	ytplayerConfigRegexp = regexp.MustCompile(`;ytplayer.config\s*=\s*({[^\n]+?});ytplayer.load`)
)

// Parser return instance
type Parser struct {
	ID            string
	VideoPageData []byte
}

// VideoInfo contains video info
type VideoInfo struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Duration string                 `json:"duration"`
	Author   string                 `json:"author"`
	DashURL  string                 `json:"dashUrl,omitempty"`
	HlsURL   string                 `json:"hlsUrl,omitempty"`
	Streams  map[string]*StreamItem `json:"streams"`
}

// NewParser create Parser instance
func NewParser(id string) (*Parser, error) {
	var (
		videoPageURL = fmt.Sprintf(videoPageHost, id)
		urls         = []string{
			videoPageURL,
		}
	)
	response, err := request.GetURLBody(urls)
	if err != nil {
		return nil, err
	}
	return &Parser{
		id,
		response[videoPageURL],
	}, nil
}

// Parse parse video info
func (p *Parser) Parse() (*VideoInfo, error) {
	var (
		info                  = &VideoInfo{ID: p.ID, Streams: make(map[string]*StreamItem)}
		ytplayerConfigMatches = ytplayerConfigRegexp.FindSubmatch(p.VideoPageData)
	)
	if len(ytplayerConfigMatches) < 2 {
		return info, fmt.Errorf("not found")
	}
	res := gjson.ParseBytes(ytplayerConfigMatches[1])
	args := res.Get("args")
	playerURL := baseURL + res.Get("assets.js").String()
	body, err := request.GetURLData(playerURL)
	if err != nil {
		return nil, err
	}
	if v := args.Get("player_response").String(); v != "" {
		if err = playerJSONParse(info, body, gjson.Parse(v)); err != nil {
			return info, err
		}
	}
	if err := fmtStreamMap(info, body, args.Get("url_encoded_fmt_stream_map").String()); err != nil {
		return info, err
	}
	if err := fmtStreamMap(info, body, args.Get("adaptive_fmts").String()); err != nil {
		return info, err
	}
	return info, nil
}

func fmtStreamMap(v *VideoInfo, body []byte, urlEncodedFmtStreamMap string) error {
	var playerjs = string(body)
	for _, s := range strings.Split(urlEncodedFmtStreamMap, ",") {
		if s == "" {
			return nil
		}
		stream, err := url.ParseQuery(s)
		if err != nil {
			return err
		}
		var itag = stream.Get("itag")
		var streamType = stream.Get("type")
		var quality = stream.Get("quality_label")
		if quality == "" {
			if v := stream.Get("qualityLabel"); v != "" {
				quality = v
			}
			if quality == "" {
				quality = stream.Get("quality")
			}
		}
		var contentLength = stream.Get("clen")
		if contentLength == "" {
			contentLength = stream.Get("contentLength")
		}
		url, err := getDownloadURL(stream, playerjs)
		if err != nil {
			return err
		}
		v.Streams[itag] = &StreamItem{
			quality,
			streamType,
			url,
			itag,
			contentLength,
		}
	}
	return nil
}

func playerJSONParse(v *VideoInfo, body []byte, res gjson.Result) error {
	var playerjs = string(body)
	var detail = res.Get("videoDetails")
	v.Title = detail.Get("title").String()
	v.Duration = detail.Get("lengthSeconds").String()
	v.Author = detail.Get("author").String()
	var streamingData = res.Get("streamingData")
	if t := streamingData.Get("dashManifestUrl").String(); t != "" {
		v.DashURL = t
	}
	if t := streamingData.Get("hlsManifestUrl").String(); t != "" {
		v.HlsURL = t
	}
	var handle = func(key gjson.Result, value gjson.Result) bool {
		var itag = value.Get("itag").String()
		var streamType = value.Get("mimeType").String()
		var quality = value.Get("qualityLabel").String()
		if quality == "" {
			quality = value.Get("quality").String()
		}
		var contentLength = value.Get("contentLength").String()

		realURL := value.Get("url").String()
		if realURL == "" {
			stream, err := url.ParseQuery(value.Get("cipher").String())
			if err != nil {
				fmt.Println(err)
			}
			realURL, err = getDownloadURL(stream, playerjs)
			if err != nil {
				fmt.Println(err)
			}
		}
		v.Streams[itag] = &StreamItem{
			quality,
			streamType,
			realURL,
			itag,
			contentLength,
		}
		return true
	}
	streamingData.Get("formats").ForEach(handle)
	streamingData.Get("adaptiveFormats").ForEach(handle)
	return nil
}
