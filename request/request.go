package request

import (
	"io/ioutil"
	"net/http"
	"sync"
	"time"
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
		age:  time.Hour,
	}
	pagecache = &bytecache{
		data: make(map[string]cacheItem),
		age:  time.Hour * 48,
	}
)

func (by *bytecache) get(url string) ([]byte, error) {
	var bs = by.cget(url)
	if bs != nil {
		return bs, nil
	}
	res, err := GetURLBody([]string{url})
	if err != nil {
		return nil, err
	}
	bs = res[url]
	by.set(url, bs)
	return bs, nil
}

func (by *bytecache) cget(key string) []byte {
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

// GetURLData check cache and get from url
func GetURLData(url string, long bool) ([]byte, error) {
	if long {
		return playercache.get(url)
	}
	return pagecache.get(url)
}

// GetURLBody run quick get
func GetURLBody(urls []string) (map[string][]byte, error) {
	type resItem struct {
		bytes []byte
		url   string
		err   error
	}
	var (
		ch       = make(chan *resItem)
		method   = http.MethodGet
		timeout  = 15
		client   = &http.Client{Timeout: time.Duration(timeout) * time.Second}
		response = make(map[string][]byte)
	)
	for _, u := range urls {
		go func(url string) {
			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				ch <- &resItem{
					nil,
					url,
					err,
				}
				return
			}
			resp, err := client.Do(req)
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
