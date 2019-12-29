package youtubevideoparser

// StreamItem is one stream
type StreamItem struct {
	Quality       string `json:"quality"`
	Type          string `json:"type"`
	URL           string `json:"url"`
	Itag          string `json:"itag"`
	ContentLength string `json:"len"`
}
