package request

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

var (
	headers = http.Header{
		"User-Agent":      []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.75 Safari/537.36"},
		"Accept-Language": []string{"zh-CN,zh;q=0.9,en;q=0.8"},
	}
)

type cacheItem struct {
	data []byte
	age  time.Time
}

type bytecache struct {
	sync.RWMutex
	data map[string]cacheItem
	age  time.Duration
}

var (
	playercache = &bytecache{
		data: make(map[string]cacheItem),
		age:  time.Hour * 48,
	}
	pagecache = &bytecache{
		data: make(map[string]cacheItem),
		age:  time.Hour,
	}
)

func (by *bytecache) geturl(url string, client http.Client) ([]byte, error) {
	var bs = by.get(url)
	if bs != nil {
		return bs, nil
	}
	res, err := GetURLBody([]string{url}, client)
	if err != nil {
		return nil, err
	}
	bs = res[url]
	by.set(url, bs)
	return bs, nil
}

func (by *bytecache) get(key string) []byte {
	by.RLock()
	item := by.data[key]
	by.RUnlock()
	if item.age.After(time.Now()) {
		return item.data
	}
	by.expire()
	return nil
}

func (by *bytecache) set(key string, data []byte) {
	by.Lock()
	by.data[key] = cacheItem{data, time.Now().Add(by.age)}
	by.Unlock()
}

func (by *bytecache) expire() {
	t := time.Now()
	by.Lock()
	for key, item := range by.data {
		if item.age.Before(t) {
			delete(by.data, key)
		}
	}
	by.Unlock()
}

// Set cache data
func Set(key string, data []byte) {
	playercache.set(key, data)
}

// Get cache data
func Get(key string) []byte {
	return playercache.get(key)
}

// GetURLData check cache and get from url
func GetURLData(url string, long bool, client http.Client) ([]byte, error) {
	if long {
		return playercache.geturl(url, client)
	}
	return pagecache.geturl(url, client)
}

// GetURLBody run quick get
func GetURLBody(urls []string, client http.Client) (map[string][]byte, error) {
	type resItem struct {
		bytes []byte
		url   string
		err   error
	}
	var (
		ch       = make(chan *resItem)
		response = make(map[string][]byte)
	)
	for _, u := range urls {
		go func(url string) {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				ch <- &resItem{
					nil,
					url,
					err,
				}
				return
			}
			req.Header = headers
			resp, err := client.Do(req)
			if err == nil {
				if resp.StatusCode != http.StatusOK {
					err = fmt.Errorf("%s:%s", url, resp.Status)
				}
			}
			if err != nil {
				ch <- &resItem{
					nil,
					url,
					err,
				}
				return
			}
			defer resp.Body.Close()
			bytes, err := ioutil.ReadAll(resp.Body)
			ch <- &resItem{
				bytes,
				url,
				err,
			}
		}(u)
	}
	for range urls {
		item := <-ch
		if item.err != nil {
			return response, item.err
		}
		response[item.url] = item.bytes
	}
	return response, nil
}
