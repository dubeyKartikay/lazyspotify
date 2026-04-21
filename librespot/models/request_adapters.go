package models

import "encoding/json"

func marshalRequest[T any](request T) ([]byte, error) {
	return json.Marshal(request)
}

func NewPlayRequest(uri string, skipToUri string, paused bool) ([]byte, error) {
	playRequest := PlayRequest{
		Uri:       uri,
		SkipToUri: skipToUri,
		Paused:    paused,
	}
	return marshalRequest(playRequest)
}

func NewSeekRequest(position int, relative bool) ([]byte, error) {
	seekRequest := SeekRequest{
		Position: position,
		Relative: relative,
	}
	return marshalRequest(seekRequest)
}

func NewVolumeRequest(volume int, relative bool) ([]byte, error) {
	volumeRequest := VolumeRequest{
		Volume:   volume,
		Relative: relative,
	}
	return marshalRequest(volumeRequest)
}

func NewShuffleRequest(shuffle bool) ([]byte, error) {
	shuffleRequest := ShuffleRequest{
		Shuffle: shuffle,
	}
	return marshalRequest(shuffleRequest)
}
