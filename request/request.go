package request

import (
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type bytecache struct {
	sync.RWMutex
	data map[string][]byte
}

var (
	bytecacher = &bytecache{
		data: make(map[string][]byte),
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
	bs := by.data[key]
	by.RUnlock()
	return bs
}

func (by *bytecache) set(key string, data []byte) {
	by.Lock()
	by.data[key] = data
	by.Unlock()
}

// GetURLData check cache and get from url
func GetURLData(url string) ([]byte, error) {
	return bytecacher.get(url)
}

// Get from memory
func Get(key string) []byte {
	return bytecacher.cget(key)
}

// Set to memory
func Set(key string, data []byte) {
	bytecacher.set(key, data)
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
