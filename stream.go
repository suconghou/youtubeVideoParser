package youtubevideoparser

// StreamItem is one stream
type StreamItem struct {
	Quality       string     `json:"quality"`
	Type          string     `json:"type"`
	URL           string     `json:"url,omitempty"`
	Itag          string     `json:"itag"`
	ContentLength string     `json:"len"`
	InitRange     *rangeItem `json:"initRange"`
	IndexRange    *rangeItem `json:"indexRange"`
}

type rangeItem struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type Caption struct {
	URL      string `json:"url,omitempty"`
	Language string `json:"language"`
}
