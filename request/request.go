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

	headers_ = http.Header{
		"User-Agent":      []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36"},
		"Accept-Language": []string{"zh-CN,zh;q=0.9,en;q=0.8"},
		"Content-Type":    []string{"application/json"},
	}

	bufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 32*1024))
		},
	}

	errTimeout = errors.New("timeout")

	httpProvider          = NewLockGeter(time.Hour * 2)
	httpProviderLongCache = NewLockGeter(time.Hour * 48)
)

const api = "https://youtubei.googleapis.com/youtubei/v1/player?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"

// LockGeter for http cache & lock get
type LockGeter struct {
	time   time.Time
	cache  time.Duration
	caches sync.Map
}

type cacheItem struct {
	time   time.Time
	ctx    context.Context
	cancel context.CancelFunc
	data   *bytes.Buffer
	err    error
}

// NewLockGeter create new lockgeter
func NewLockGeter(cache time.Duration) *LockGeter {
	return &LockGeter{
		time:   time.Now(),
		cache:  cache,
		caches: sync.Map{},
	}
}

// Get with lock & cache,the return bytes is readonly
func (l *LockGeter) DoRequest(url string, method string, reqHeaders http.Header, body io.Reader, cackeKey string, client http.Client) ([]byte, error) {
	var now = time.Now()
	l.clean(now)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t, loaded := l.caches.LoadOrStore(cackeKey, &cacheItem{
		time:   now,
		ctx:    ctx,
		cancel: cancel,
		err:    errTimeout,
	})
	v := t.(*cacheItem)
	if loaded {
		<-v.ctx.Done()
		if v.data == nil {
			return nil, v.err
		}
		return v.data.Bytes(), v.err
	}
	data, err := DoRequest(url, method, reqHeaders, body, client)
	v.data = data
	v.err = err
	cancel()
	if data == nil {
		return nil, err
	}
	return data.Bytes(), err
}

func (l *LockGeter) clean(now time.Time) {
	if now.Sub(l.time) < time.Second*5 {
		return
	}
	l.caches.Range(func(key, value interface{}) bool {
		var v = value.(*cacheItem)
		if now.Sub(v.time) > l.cache {
			v.cancel()
			if v.data != nil {
				v.data.Reset()
				bufferPool.Put(v.data)
			}
			l.caches.Delete(key)
		}
		return true
	})
	l.time = now
}

// LockGeter的调用都有bufferPool.Put,外部调用即时没有bufferPool.Put也不会内存泄露
func DoRequest(url string, method string, reqHeaders http.Header, body io.Reader, client http.Client) (*bytes.Buffer, error) {
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
	var buffer = bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	_, err = buffer.ReadFrom(resp.Body)
	if err != nil {
		buffer.Reset()
		bufferPool.Put(buffer)
		return nil, err
	}
	return buffer, nil
}

func CacheGet(url string, client http.Client) ([]byte, error) {
	return httpProvider.DoRequest(url, http.MethodGet, headers, nil, url, client)
}

func CacheGetLong(url string, client http.Client) ([]byte, error) {
	return httpProviderLongCache.DoRequest(url, http.MethodGet, headers, nil, url, client)
}

func CachePost(id string, client http.Client) ([]byte, error) {
	var body = strings.NewReader(`{"videoId":"` + id + `","context":{"client":{"clientName":"Android","clientVersion":"16.13.35"}}}`)
	return httpProvider.DoRequest(api, http.MethodPost, headers_, body, id, client)
}
