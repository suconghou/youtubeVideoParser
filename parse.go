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
	)
	videoPageData, err := request.GetURLData(videoPageURL, false)
	if err != nil {
		return nil, err
	}
	return &Parser{
		id,
		videoPageData,
	}, nil
}

// Parse parse video info
func (p *Parser) Parse() (*VideoInfo, error) {
	var (
		info                  = &VideoInfo{ID: p.ID, Streams: make(map[string]*StreamItem)}
		ytplayerConfigMatches = ytplayerConfigRegexp.FindSubmatch(p.VideoPageData)
	)
	if len(ytplayerConfigMatches) < 2 {
		// if page parse failed, we try api parse again
		return parse(info)
	}
	res := gjson.ParseBytes(ytplayerConfigMatches[1])
	args := res.Get("args")
	playerURL := baseURL + res.Get("assets.js").String()
	body, err := request.GetURLData(playerURL, true)
	if err != nil {
		return nil, err
	}
	if v := args.Get("player_response").String(); v != "" {
		if err = playerJSONParse(info, body, gjson.Parse(v)); err != nil {
			return info, err
		}
	}
	return info, nil
}

func parse(v *VideoInfo) (*VideoInfo, error) {
	var (
		videoInfoURL = fmt.Sprintf(videoInfoHost, v.ID)
	)
	videoInfoData, err := request.GetURLData(videoInfoURL, false)
	if err != nil {
		return v, err
	}
	values, err := url.ParseQuery(string(videoInfoData))
	if err != nil {
		return v, err
	}
	status := values.Get("status")
	if status != "ok" {
		return v, fmt.Errorf("%s %s:%s", status, values.Get("errorcode"), values.Get("reason"))
	}
	res := values.Get("player_response")
	err = playerJSONParse(v, nil, gjson.Parse(res))
	return v, err
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
	streamingData.Get("formats").ForEach(handle)
	streamingData.Get("adaptiveFormats").ForEach(handle)
	return nil
}
