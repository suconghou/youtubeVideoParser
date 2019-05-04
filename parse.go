package youtubeVideoParser

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/suconghou/youtubeVideoParser/request"
)

// Parser return instance
type Parser struct {
	ID            string
	VideoPageData []byte
	VideoInfoData []byte
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
		videoInfoURL = fmt.Sprintf(videoInfoHost, id)
		urls         = []string{
			videoPageURL,
			videoInfoURL,
		}
	)
	response, err := request.GetURLBody(urls)
	if err != nil {
		return nil, err
	}
	return &Parser{
		id,
		response[videoPageURL],
		response[videoInfoURL],
	}, nil
}

// Parse parse video info
func (p *Parser) Parse() (*VideoInfo, error) {
	var (
		info        = &VideoInfo{ID: p.ID, Streams: make(map[string]*StreamItem)}
		values, err = url.ParseQuery(string(p.VideoInfoData))
	)
	if err != nil {
		return info, err
	}
	if ytplayerConfigMatches := ytplayerConfigRegexp.FindSubmatch(p.VideoPageData); len(ytplayerConfigMatches) > 1 {
		js, err := simplejson.NewJson(ytplayerConfigMatches[1])
		if err != nil {
			return info, err
		}
		info.Title = js.GetPath("args", "title").MustString()
		info.Duration = js.GetPath("args", "length_seconds").MustString()
		info.Author = js.GetPath("args", "author").MustString()
		if err = fmtStreamMap(info, js.GetPath("args", "url_encoded_fmt_stream_map").MustString()); err != nil {
			return info, err
		}
		if err = fmtStreamMap(info, js.GetPath("args", "adaptive_fmts").MustString()); err != nil {
			return info, err
		}
	}
	status := values.Get("status")
	if status == "ok" {
		info.Title = values.Get("title")
		info.Duration = values.Get("length_seconds")
		info.Author = values.Get("author")
		if err = fmtStreamMap(info, values.Get("url_encoded_fmt_stream_map")); err != nil {
			return info, err
		}
		if err = fmtStreamMap(info, values.Get("adaptive_fmts")); err != nil {
			return info, err
		}

	}
	if status == "fail" {
		errorcode := values.Get("errorcode")
		if errorcode == "2" {
			// Invalid parameters
			return info, fmt.Errorf(values.Get("reason"))
		}
		if errorcode == "150" {
			return info, fmt.Errorf(values.Get("reason"))
		}
	}
	return info, nil
}

func fmtStreamMap(v *VideoInfo, urlEncodedFmtStreamMap string) error {
	for _, item := range strings.Split(urlEncodedFmtStreamMap, ",") {
		if item == "" {
			return nil
		}
		streamMap, err := url.ParseQuery(item)
		if err != nil {
			return err
		}
		fmt.Println(streamMap)
		var quality = streamMap.Get("quality")
		if quality == "" {
			quality = streamMap.Get("quality_label")
		}
		v.Streams[streamMap.Get("itag")] = &StreamItem{
			quality,
			streamMap.Get("type"),
			streamMap.Get("url"),
			streamMap.Get("itag"),
		}
	}
	return nil
}

// Parse parse video info by id
/**



// GetYoutubeImageURL return img url
func GetYoutubeImageURL(id string, imgtype string, quality string) string {
	var (
		host           string
		qa             string
		defaultType    = "webp"
		defaultQuality = "medium"
	)
	if v, ok := youtubeImageHostMap[imgtype]; ok {
		host = v
	} else {
		host = youtubeImageHostMap[defaultType]
	}
	if v, ok := youtubeImageMap[quality]; ok {
		qa = v
	} else {
		qa = youtubeImageMap[defaultQuality]
	}
	return fmt.Sprintf("%s%s/%s.%s", host, id, qa, imgtype)
}

// GetYoutubeVideoURL return video url
func GetYoutubeVideoURL(id string, videotype string, quality string) (string, error) {
	info, err := Parse(id)
	if err != nil {
		return "", err
	}
	return info.MustGetVideoURL(videotype, quality), nil
}



**/
