package youtubeVideoParser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	qualityHIGHRES   = "highres"
	qualityHD1080    = "hd1080"
	qualityHD720     = "hd720"
	qualityLARGE     = "large"
	qualityMEDIUM    = "medium"
	qualitySMALL     = "small"
	formatWEBM       = "webm"
	formatMP4        = "mp4"
	formatFLV        = "flv"
	format3GP        = "3gp"
	videoPageHost    = "https://www.youtube.com/watch?v=%s"
	youtubeVideoHost = "https://www.youtube.com/get_video_info?video_id=%s"
	html5PlayerHost  = "https://www.youtube.com%s"
)

var (
	ytplayerConfigRegexp = regexp.MustCompile(`ytplayer.config\s*=\s*([^\n]+?});`)
	youtubeImageMap      = map[string]string{
		"large":  "hqdefault",
		"medium": "mqdefault",
		"small":  "default",
	}
	youtubeImageHostMap = map[string]string{
		"jpg":  "http://i.ytimg.com/vi/",
		"webp": "http://i.ytimg.com/vi_webp/",
	}
	mimeMap = map[string]string{
		"video/3gpp":  format3GP,
		"video/mp4":   formatMP4,
		"video/webm":  formatWEBM,
		"video/x-flv": formatFLV,
		"audio/webm":  formatWEBM,
		"audio/mp4":   formatMP4,
	}
	videoTypes = map[string]string{
		formatWEBM: "video/webm",
		formatMP4:  "video/mp4",
		formatFLV:  "video/x-flv",
		format3GP:  "video/3gpp",
	}
	sortedQualities = []string{
		qualityHIGHRES,
		qualityHD1080,
		qualityHD720,
		qualityLARGE,
		qualityMEDIUM,
		qualitySMALL,
	}
	sortedFormats = []string{
		formatWEBM,
		formatMP4,
		formatFLV,
		format3GP,
	}
)

// StreamItem is one stream
type StreamItem struct {
	Quality   string
	Type      string
	URL       string
	Itag      string
	Mime      string
	Container string
}

// VideoInfo is a video info
type VideoInfo struct {
	ID           string
	Title        string
	Duration     string
	Keywords     string
	Author       string
	DefaultStram *StreamItem
	Streams      map[string]*StreamItem
}

// Stringify return video info []byte string
func (v *VideoInfo) Stringify() ([]byte, error) {
	bs, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

// GetVideoItem return one url or error
func (v *VideoInfo) GetVideoItem(ext string, quality string) (*StreamItem, error) {
	vtype := videoTypes[ext]
	if vtype != "" {
		for _, item := range v.Streams {
			if item.Quality == quality && item.Mime == vtype && strings.Contains(item.Type, ",") { // have audio and video
				return item, nil
			}
		}
	}
	return nil, fmt.Errorf("not found type %s quality %s", ext, quality)
}

// MustGetVideoURL must return one url
func (v *VideoInfo) MustGetVideoURL(ext string, quality string) string {
	u, err := v.GetVideoItem(ext, quality)
	if err != nil {
		for _, q := range sortedQualities {
			u, err = v.GetVideoItem(ext, q)
			if err == nil {
				return u.URL
			}
		}
		for _, f := range sortedFormats {
			u, err = v.GetVideoItem(f, quality)
			if err == nil {
				return u.URL
			}
		}
		for _, f := range sortedFormats {
			for _, q := range sortedQualities {
				u, err = v.GetVideoItem(f, q)
				if err == nil {
					return u.URL
				}
			}
		}
	}
	return u.URL
}
