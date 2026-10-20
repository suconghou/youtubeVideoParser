package request

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	headers = http.Header{
		"User-Agent":      []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36"},
		"Accept-Language": []string{"zh-CN,zh;q=0.9,en;q=0.8"},
	}

	bufferPool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 32*1024))
		},
	}

	errTimeout = errors.New("timeout")

	HttpProvider = NewLockGeter()
)

const api = "https://www.youtube.com/youtubei/v1/player?key=AIzaSyB-63vPrdThhKuerbB2N_l7Kwwcxj6yUAc&prettyPrint=false"

// clientConfig 定义 YouTube InnerTube client 请求参数
type clientConfig struct {
	name    string // clientName
	version string // clientVersion
	ua      string // User-Agent 头
	headers http.Header
	body    func(videoID string) string
}

// androidVR 是 ANDROID_VR client（实测 26/26 直链，最快）
func androidVR(videoID string) string {
	return `{"context":{"client":{"clientName":"ANDROID_VR","clientVersion":"1.65.10","deviceMake":"Oculus","deviceModel":"Quest 3","androidSdkVersion":32,"osName":"Android","osVersion":"12L","hl":"en","gl":"US"}},"videoId":"` + videoID + `","contentCheckOk":true,"racyCheckOk":true}`
}

// ios 是 IOS client（实测 27/27 直链，备选）
func ios(videoID string) string {
	return `{"context":{"client":{"clientName":"IOS","clientVersion":"21.02.3","deviceModel":"iPhone16,2","osName":"iOS","osVersion":"18.1.0","hl":"en","gl":"US"}},"videoId":"` + videoID + `","contentCheckOk":true,"racyCheckOk":true}`
}

var clients = []clientConfig{
	{
		name:    "ANDROID_VR",
		version: "1.65.10",
		ua:      "com.google.android.apps.youtube.vr.oculus/1.65.10 (Linux; U; Android 12L; eureka-user Build/SQ3A.220605.009.A1) gzip",
		headers: http.Header{
			"Content-Type":             []string{"application/json"},
			"X-Youtube-Client-Name":    []string{"28"},
			"X-Youtube-Client-Version": []string{"1.65.10"},
			"Origin":                   []string{"https://www.youtube.com"},
			"Accept-Language":          []string{"en-US,en;q=0.9"},
		},
		body: androidVR,
	},
	{
		name:    "IOS",
		version: "21.02.3",
		ua:      "com.google.ios.youtube/21.02.3 (iPhone16,2; U; CPU iOS 18_1_0 like Mac OS X;)",
		headers: http.Header{
			"Content-Type":             []string{"application/json"},
			"X-Youtube-Client-Name":    []string{"5"},
			"X-Youtube-Client-Version": []string{"21.02.3"},
			"Origin":                   []string{"https://www.youtube.com"},
			"Accept-Language":          []string{"en-US,en;q=0.9"},
		},
		body: ios,
	},
}

// LockGeter for http cache & lock get
type LockGeter struct {
	time   int64
	caches sync.Map
}

type cacheItem struct {
	time    int64
	ctx     context.Context
	cancel  context.CancelFunc
	data    []byte // immutable, safe to return to callers
	err     error
	loading bool
}

// NewLockGeter create new lockgeter
func NewLockGeter() *LockGeter {
	return &LockGeter{
		time:   0,
		caches: sync.Map{},
	}
}

// Get with lock & cache,the return bytes is readonly
func (l *LockGeter) DoRequest(url string, method string, reqHeaders http.Header, body io.Reader, cackeKey string, client http.Client, ttl int64) ([]byte, error) {
	var now = time.Now().Unix()
	l.clean(now)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t, loaded := l.caches.LoadOrStore(cackeKey, &cacheItem{
		time:    now + ttl,
		ctx:     ctx,
		cancel:  cancel,
		err:     errTimeout,
		loading: true,
	})
	v := t.(*cacheItem)
	if loaded {
		<-v.ctx.Done()
		v.loading = false
		if v.data == nil {
			return nil, v.err
		}
		return v.data, v.err
	}
	data, err := doRequestBytes(url, method, reqHeaders, body, client)
	v.data = data
	v.err = err
	v.loading = false
	cancel()
	if data == nil {
		return nil, err
	}
	return data, err
}

func (l *LockGeter) clean(now int64) {
	if now-l.time < 5 {
		return
	}
	l.time = now
	l.caches.Range(func(key, value any) bool {
		var v = value.(*cacheItem)
		if v.time < now && !v.loading {
			v.cancel()
			l.caches.Delete(key)
		}
		return true
	})
}

// DoRequest http data, the returned buffer is caller-owned and readonly.
func DoRequest(url string, method string, reqHeaders http.Header, body io.Reader, client http.Client) (*bytes.Buffer, error) {
	data, err := doRequestBytes(url, method, reqHeaders, body, client)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(bytes.Clone(data)), nil
}

// doRequestBytes http data, the return value is immutable and can be held forever.
func doRequestBytes(url string, method string, reqHeaders http.Header, body io.Reader, client http.Client) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header = reqHeaders
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s:%s", url, resp.Status)
	}
	var buf = bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		buf.Reset()
		bufferPool.Put(buf)
		return nil, err
	}
	data := bytes.Clone(buf.Bytes())
	buf.Reset()
	bufferPool.Put(buf)
	return data, nil
}

func CacheGet(url string, client http.Client) ([]byte, error) {
	return HttpProvider.DoRequest(url, http.MethodGet, headers, nil, url, client, 7200)
}

func CacheGetLong(url string, client http.Client) ([]byte, error) {
	return HttpProvider.DoRequest(url, http.MethodGet, headers, nil, url, client, 86400)
}

// CachePost 依次尝试 ANDROID_VR → IOS client 直到拿到响应
func CachePost(id string, client http.Client) ([]byte, error) {
	var errs = []string{}
	for i, c := range clients {
		var h = c.headers.Clone()
		h.Set("User-Agent", c.ua)
		var key = fmt.Sprintf("%s/%s", id, c.name)
		bs, err := HttpProvider.DoRequest(api, http.MethodPost, h, strings.NewReader(c.body(id)), key, client, 7200)
		if err == nil {
			return bs, nil
		}
		errs = append(errs, fmt.Sprintf("%s_%s:%s", c.name, c.version, err))
		if i == 0 {
			continue
		}
	}
	return nil, errors.New(strings.Join(errs, ";"))
}
