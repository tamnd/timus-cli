package timus

// Problem is a problem from Timus Online Judge.
type Problem struct {
	Rank       int    `json:"rank"`
	Num        int    `json:"num"`
	Title      string `json:"title"`
	Solved     int    `json:"solved"`
	Difficulty int    `json:"difficulty"`
	URL        string `json:"url"`
}
