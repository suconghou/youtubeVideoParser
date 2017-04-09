package youtubeVideoParser

import (
	"encoding/json"
	"fmt"
	_ "strings"
)

const (
	QUALITY_HIGHRES = "highres"
	QUALITY_HD1080  = "hd1080"
	QUALITY_HD720   = "hd720"
	QUALITY_LARGE   = "large"
	QUALITY_MEDIUM  = "medium"
	QUALITY_SMALL   = "small"

	FORMAT_WEBM = "webm"
	FORMAT_MP4  = "mp4"
	FORMAT_FLV  = "flv"
	FORMAT_3GP  = "3gp"
)

var mapedQualities map[string]string = map[string]string{
	QUALITY_HIGHRES: QUALITY_LARGE,
	QUALITY_HD1080:  QUALITY_LARGE,
	QUALITY_HD720:   QUALITY_LARGE,
	QUALITY_LARGE:   QUALITY_LARGE,
	QUALITY_MEDIUM:  QUALITY_MEDIUM,
	QUALITY_SMALL:   QUALITY_SMALL,
}

var mimeMap map[string]string = map[string]string{
	"video/3gpp":  FORMAT_3GP,
	"video/mp4":   FORMAT_MP4,
	"video/webm":  FORMAT_WEBM,
	"video/x-flv": FORMAT_FLV,
}

var formatsTrigger map[string]string = map[string]string{
	FORMAT_WEBM: "video/webm",
	FORMAT_MP4:  "video/mp4",
	FORMAT_FLV:  "video/x-flv",
	FORMAT_3GP:  "video/3gpp",
}

var largeTypes []string = []string{
	QUALITY_HIGHRES,
	QUALITY_HD1080,
	QUALITY_HD720,
}

var mediumTypes []string = []string{
	QUALITY_MEDIUM,
}

var smallTypes []string = []string{
	QUALITY_SMALL,
}

var sortedFormats []string = []string{
	FORMAT_WEBM,
	FORMAT_MP4,
	FORMAT_FLV,
	FORMAT_3GP,
}

type streamItem struct {
	Itag      string
	Url       string
	Sig       string
	S         string
	Quality   string
	Type      string
	Mime      string
	Container string
}

type videoInfo struct {
	Id       string
	Title    string
	Duration string
	Keywords string
	Author   string
	Stream   map[string]streamItem
}

func (info videoInfo) ToJson() ([]byte, error) {
	if bs, err := json.Marshal(&info); err != nil {
		return []byte(""), err
	} else {
		return bs, nil
	}
}

func (info videoInfo) GetStream(quality string, preferType string) (string, string, error) {
	if quality == QUALITY_LARGE {
		url, t, err := info.GetSuchStream(QUALITY_LARGE, preferType)
		if err != nil {
			url, t, err = info.GetSuchStream(QUALITY_MEDIUM, preferType)
			if err != nil {
				url, t, err = info.GetSuchStream(QUALITY_SMALL, preferType)
			}
		}
		return url, t, err
	} else if quality == QUALITY_MEDIUM {
		url, t, err := info.GetSuchStream(QUALITY_MEDIUM, preferType)
		if err != nil {
			url, t, err = info.GetSuchStream(QUALITY_LARGE, preferType)
			if err != nil {
				url, t, err = info.GetSuchStream(QUALITY_SMALL, preferType)
			}
		}
		return url, t, err
	} else {
		url, t, err := info.GetSuchStream(QUALITY_SMALL, preferType)
		if err != nil {
			url, t, err = info.GetSuchStream(QUALITY_MEDIUM, preferType)
			if err != nil {
				url, t, err = info.GetSuchStream(QUALITY_LARGE, preferType)
			}
		}
		return url, t, err
	}
}

func (info videoInfo) GetSuchStream(quality string, preferType string) (string, string, error) {
	var urls map[string]string = map[string]string{}
	for _, item := range info.Stream {
		if item.Quality == quality {
			if item.Container == preferType {
				return item.Url, item.Container, nil
			} else {
				urls[item.Container] = item.Url
			}
		} else {
			continue
		}
	}
	for _, t := range sortedFormats {
		if v, ok := urls[t]; ok {
			return v, t, nil
		}
	}
	return "", "", fmt.Errorf("not found such stream")
}
