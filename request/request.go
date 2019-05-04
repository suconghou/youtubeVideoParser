package request

import (
	"io/ioutil"
	"net/http"
	"time"
)

// GetURLBody run quick get
func GetURLBody(urls []string) (map[string][]byte, error) {
	type resItem struct {
		bytes []byte
		url   string
		err   error
	}
	var (
		ch       = make(chan *resItem)
		method   = "GET"
		timeout  = 5
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
