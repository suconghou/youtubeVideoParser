package youtubeVideoParser

import (
	"encoding/json"
	"fmt"
	"strings"
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
		if item["quality"] == quality {
			if item["type"] == preferType {
				return item["url"], item["type"], nil
			} else {
				urls[item["type"]] = item["url"]
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

func tUrl(url string, sig string) string {
	return url + "&signature=" + sig
}

func tFormat(types string) string {
	for format, trigger := range formatsTrigger {
		if strings.Contains(types, trigger) {
			return format
		}
	}
	return FORMAT_MP4
}

func tQuality(qualitystr string) string {
	if quality, ok := mapedQualities[qualitystr]; ok {
		return quality
	}
	return QUALITY_SMALL
}
