package db

import "time"

type VideoParams struct {
	Title    string `json:"title"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Duration int    `json:"duration"`
	Thumb    string `json:"thumb"`
}

type Metadata struct {
	VideoID   string
	Quality   string
	Mime      string
	FileID    string
	Params    VideoParams
	CreatedAt time.Time
	UpdatedAt time.Time
}
