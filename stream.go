package youtubeVideoParser

import (
	"encoding/json"
	"strings"
)

const (
	QUALITY_HIGHRES = "highres"
	QUALITY_HD1080  = "hd1080"
	QUALITY_HD720   = "hd720"
	QUALITY_LARGE   = "large"
	QUALITY_MEDIUM  = "medium"
	QUALITY_SMALL   = "small"
	QUALITY_MIN     = "min"
	QUALITY_MAX     = "max"
	QUALITY_UNKNOWN = "unknown"

	FORMAT_MP4     = "mp4"
	FORMAT_WEBM    = "webm"
	FORMAT_FLV     = "flv"
	FORMAT_3GP     = "3ggp"
	FORMAT_UNKNOWN = "unknown"
)

var sortedQualities []string = []string{
	QUALITY_HIGHRES,
	QUALITY_HD1080,
	QUALITY_HD720,
	QUALITY_LARGE,
	QUALITY_MEDIUM,
	QUALITY_SMALL,
	QUALITY_UNKNOWN,
}

var formatsTrigger map[string]string = map[string]string{
	FORMAT_MP4:  "video/mp4",
	FORMAT_FLV:  "video/x-flv",
	FORMAT_WEBM: "video/webm",
	FORMAT_3GP:  "video/3gpp",
}

var sortedFormats []string = []string{
	FORMAT_MP4,
	FORMAT_FLV,
	FORMAT_WEBM,
	FORMAT_3GP,
	FORMAT_UNKNOWN,
}

type stream map[string]string
type streamList []stream

type videoInfo struct {
	Id       string
	Title    string
	Duration string
	Keywords string
	Author   string
	Stream   streamList
}

func (s stream) Url() string {
	return s["url"] + "&signature=" + s["sig"]
}

func (s stream) Format() string {
	for format, trigger := range formatsTrigger {
		if strings.Contains(s["type"], trigger) {
			return format
		}
	}
	return FORMAT_UNKNOWN
}

func (s stream) Quality() string {
	for _, quality := range sortedQualities {
		if quality == s["quality"] {
			return quality
		}
	}
	return QUALITY_UNKNOWN
}

func (info videoInfo) ToJson() ([]byte, error) {
	if bs, err := json.Marshal(&info); err != nil {
		return []byte(""), err
	} else {
		return bs, nil
	}

}
