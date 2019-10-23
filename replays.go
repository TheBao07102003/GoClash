package clash

import "fmt"

type ReplayVersion struct {
	Major   int `json:"major"`
	Build   int `json:"build"`
	Content int `json:"content"`
}
type Replay struct {
	BattleTime string `json:"battleTime"`
	// Replay data is hideously unstructured, so let's save some time.
	ReplayData map[string]interface{} `json:"replayData"`
	ShareCount int                    `json:"shareCount"`
	Tag        string                 `json:"tag"`
	ViewCount  int                    `json:"viewCount"`
	Version    ReplayVersion          `json:"version"`
}

type ReplayService struct {
	c   *Client
	tag string
}

func (c *Client) Replay(tag string) *ReplayService {
	return &ReplayService{c, tag}
}

// Get information about a single replay by a replay tag.
func (i *ReplayService) Get() (Replay, error) {
	path := "/v1/replays/%s"
	url := fmt.Sprintf(path, NormaliseTag(i.tag))
	req, err := i.c.NewRequest("GET", url, nil)
	var replay Replay

	if err == nil {
		_, err = i.c.Do(req, &replay, path)
	}

	return replay, err
}
