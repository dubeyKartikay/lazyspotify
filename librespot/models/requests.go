package models

type PlayRequest struct {
	Uri       string `json:"uri"`
	SkipToUri string `json:"skip_to_uri"`
	Paused    bool   `json:"paused"`
}

type SeekRequest struct {
	Position int  `json:"position"`
	Relative bool `json:"relative"`
}

type VolumeRequest struct {
	Volume   int  `json:"volume"`
	Relative bool `json:"relative"`
}

type ShuffleRequest struct {
	Shuffle bool `json:"shuffle_context"`
}
