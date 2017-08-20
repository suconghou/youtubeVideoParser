package youtubeVideoParser

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/bitly/go-simplejson"
)

// Parse parse video info by id
func Parse(id string) (*VideoInfo, error) {
	u := fmt.Sprintf(videoPageHost, id)
	info := &VideoInfo{ID: id, Streams: make(map[string]*StreamItem)}
	res, err := httpGet(fmt.Sprintf(youtubeVideoHost, id))
	if err != nil {
		return info, err
	}
	values, err := url.ParseQuery(string(res))
	if err != nil {
		return info, err
	}
	status := values.Get("status")
	if status == "ok" {
		info.Title = values.Get("title")
		info.Duration = values.Get("length_seconds")
		info.Keywords = values.Get("keywords")
		info.Author = values.Get("author")
		videoPage, err := httpGet(u)
		if err != nil {
			return info, err
		}
		ytplayerConfigMatches := ytplayerConfigRegexp.FindSubmatch(videoPage)
		if len(ytplayerConfigMatches) > 1 {
			jsonstr := ytplayerConfigMatches[1]
			js, err := simplejson.NewJson(jsonstr)
			if err != nil {
				return info, err
			}
			streamstr := js.GetPath("args", "url_encoded_fmt_stream_map").MustString()
			html5player := fmt.Sprintf(html5PlayerHost, js.GetPath("assets", "js").MustString())
			streams := strings.Split(streamstr, ",")
			for _, item := range streams {
				strItem, err := getstream(item, html5player, false)
				if err != nil {
					return info, err
				}
				info.Streams[strItem.Itag] = strItem
			}
			parseMore(js, info, html5player)
		}
		return info, nil
	} else if status == "fail" {
		errorcode := values.Get("errorcode")
		reason := values.Get("reason")
		curerr := fmt.Errorf("errorcode %s:%s", errorcode, reason)
		if errorcode == "150" {
			if strings.Contains(reason, "unavailable") {
				return info, curerr
			}
			videoPage, err := httpGet(u)
			if err != nil {
				return info, err
			}
			ytplayerConfigMatches := ytplayerConfigRegexp.FindSubmatch(videoPage)
			if len(ytplayerConfigMatches) > 1 {
				jsonstr := ytplayerConfigMatches[1]
				js, err := simplejson.NewJson(jsonstr)
				if err != nil {
					return info, err
				}
				info.Title = js.GetPath("args", "title").MustString()
				info.Duration = js.GetPath("args", "length_seconds").MustString()
				info.Author = js.GetPath("args", "author").MustString()
				info.Keywords = js.GetPath("args", "keywords").MustString()
				streamstr := js.GetPath("args", "url_encoded_fmt_stream_map").MustString()
				html5player := fmt.Sprintf(html5PlayerHost, js.GetPath("assets", "js").MustString())
				streams := strings.Split(streamstr, ",")
				for _, item := range streams {
					strItem, err := getstream(item, html5player, false)
					if err != nil {
						return info, err
					}
					info.Streams[strItem.Itag] = strItem
				}
				parseMore(js, info, html5player)
			}
		} else {
			return info, curerr
		}
		return info, nil
	} else {
		return info, fmt.Errorf("error status: %s in get_video_info", status)
	}
}

func parseMore(js *simplejson.Json, info *VideoInfo, player string) {
	fmts := js.GetPath("args", "adaptive_fmts").MustString()
	if fmts != "" {
		streams := strings.Split(fmts, ",")
		for _, item := range streams {
			strItem, err := getstream(item, player, true)
			if err == nil {
				info.Streams[strItem.Itag] = strItem
			}
		}
	}
}

// GetYoutubeVideoInfo return all video info format in json
func GetYoutubeVideoInfo(id string) ([]byte, error) {
	info, err := Parse(id)
	if err != nil {
		return nil, err
	}
	return info.Stringify()
}

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

func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, err
	}
	return body, nil
}

// 两种格式的解析
func getstream(str string, player string, more bool) (*StreamItem, error) {
	if str != "" {
		values, err := url.ParseQuery(str)
		if err != nil {
			return nil, err
		}
		item := &StreamItem{
			Quality: values.Get("quality"),
			URL:     values.Get("url"),
			Type:    values.Get("type"),
			Itag:    values.Get("itag"),
		}
		mime := strings.Split(item.Type, ";")[0]
		item.Mime = mime
		item.Container = mimeMap[mime]
		if more {
			item.URL = fmt.Sprintf("%s&ratebypass=yes", item.URL) // get over speed limiting
			item.Quality = getQuality(&values)
		}
		s := values.Get("s")
		if s != "" {
			item.URL = decipher(item, s, player)
		}
		return item, nil
	}
	return nil, nil
}

func decipher(item *StreamItem, s string, player string) string {
	var (
		reverse = func(arr []string) []string {
			for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
				arr[i], arr[j] = arr[j], arr[i]
			}
			return arr
		}
		swap = func(arr []string, b int) []string {
			c := arr[0]
			arr[0] = arr[b%len(arr)]
			arr[b] = c
			return arr
		}
	)
	match := playerIDRegexp.FindStringSubmatch(player)
	var arr []string
	if len(match) > 1 {
		id := match[1]
		if id == "vflAAoWvh" { //20170820 add
			arr = reverse(strings.Split(s, ""))[1:]
			arr = reverse(arr)[2:]
			arr = reverse(arr)[2:]
		} else { // old
			arr = reverse(strings.Split(s[3:], ""))
			arr = swap(arr, 36)
			arr = swap(arr[1:], 48)
		}
	}
	return fmt.Sprintf("%s&signature=%s", item.URL, strings.Join(arr, ""))
}

func getQuality(v *url.Values) string {
	q := v.Get("quality_label")
	if q != "" { // video
		if strings.Contains(q, "144p") || strings.Contains(q, "240p") {
			return qualitySMALL
		} else if strings.Contains(q, "360p") {
			return qualityMEDIUM
		} else if strings.Contains(q, "480p") {
			return qualityLARGE
		} else if strings.Contains(q, "720p") {
			return qualityHD720
		} else if strings.Contains(q, "1080p") {
			return qualityHD1080
		}
		return qualityHIGHRES
	}
	bitrate, err := strconv.Atoi(v.Get("bitrate"))
	if err == nil { // audio
		if bitrate <= 90000 {
			return qualitySMALL
		} else if bitrate > 90000 && bitrate <= 140000 {
			return qualityMEDIUM
		}
		return qualityLARGE
	}
	return qualityHIGHRES
}
