package clash

import "fmt"

type LocationPager struct {
	Items  []Location `json:"items"`
	Paging Paging     `json:"paging"`
}

type Location struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	IsCountry   bool   `json:"isCountry"`
	CountryCode string `json:"countryCode,omitempty"`
}

type LocationClanRankingPager struct {
	Items  []ClanRanking `json:"items"`
	Paging Paging        `json:"paging"`
}

type LocationPlayerRankingPager struct {
	Items  []PlayerRanking `json:"items"`
	Paging Paging          `json:"paging"`
}

type ClanRanking struct {
	Tag          string   `json:"tag"`
	Name         string   `json:"name"`
	Rank         int      `json:"rank"`
	PreviousRank int      `json:"previousRank"`
	Location     Location `json:"location"`
	BadgeId      int      `json:"badgeId"`
	ClanScore    int      `json:"clanScore"`
	Members      int      `json:"members"`
}

type PlayerRanking struct {
	Tag          string     `json:"tag"`
	Name         string     `json:"name"`
	ExpLevel     int        `json:"expLevel"`
	Trophies     int        `json:"trophies"`
	Clan         PlayerClan `json:"clan"`
	Rank         int        `json:"Rank"`
	PreviousRank int        `json:"previousRank"`
	Arena        Arena      `json:"arena"`
}

type LocationsService struct {
	c *Client
}

type LocationService struct {
	c  *Client
	id string
}

func (c *Client) Locations() *LocationsService {
	return &LocationsService{c}
}

// NB: Location ID is a string. This is because 'global' is a valid location ID.
func (c *Client) Location(id string) *LocationService {
	return &LocationService{c, id}
}

// List all available locations
func (i *LocationsService) All() (LocationPager, error) {
	path := "/v1/locations"
	req, err := i.c.NewRequest("GET", path, nil)

	var locations LocationPager

	if err == nil {
		_, err = i.c.Do(req, &locations, path)
	}

	return locations, err
}

// Get information about specific location
func (i *LocationService) Get() (Location, error) {
	path := "/v1/locations/%s"
	req, err := i.c.NewRequest("GET", fmt.Sprintf(path, i.id), nil)

	var location Location

	if err == nil {
		_, err = i.c.Do(req, &location, path)
	}

	return location, err
}

// Get clan rankings for a specific location
func (i *LocationService) ClanRankings(query *PagedQuery) (LocationClanRankingPager, error) {
	path := "/v1/locations/%s/rankings/clans"
	req, err := i.c.NewRequest("GET", fmt.Sprintf(path, i.id), nil)

	q := req.URL.Query()

	if query.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", query.Limit))
	}

	if query.After > 0 {
		q.Add("after", fmt.Sprintf("%d", query.After))
	}

	if query.Before > 0 {
		q.Add("before", fmt.Sprintf("%d", query.Before))
	}

	req.URL.RawQuery = q.Encode()

	var rankings LocationClanRankingPager

	if err == nil {
		_, err = i.c.Do(req, &rankings, path)
	}

	return rankings, err
}

// Get player rankings for a specific location
func (i *LocationService) PlayerRankings(query *PagedQuery) (LocationPlayerRankingPager, error) {
	path := "/v1/locations/%s/rankings/players"
	req, err := i.c.NewRequest("GET", fmt.Sprintf(path, i.id), nil)

	q := req.URL.Query()

	if query.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", query.Limit))
	}

	if query.After > 0 {
		q.Add("after", fmt.Sprintf("%d", query.After))
	}

	if query.Before > 0 {
		q.Add("before", fmt.Sprintf("%d", query.Before))
	}

	req.URL.RawQuery = q.Encode()

	var rankings LocationPlayerRankingPager

	if err == nil {
		_, err = i.c.Do(req, &rankings, path)
	}

	return rankings, err
}

// Get clan war rankings for a specific location
func (i *LocationService) ClanWarRankings(query *PagedQuery) (LocationClanRankingPager, error) {
	path := "/v1/locations/%s/rankings/clanwars"
	req, err := i.c.NewRequest("GET", fmt.Sprintf(path, i.id), nil)

	q := req.URL.Query()

	if query.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", query.Limit))
	}

	if query.After > 0 {
		q.Add("after", fmt.Sprintf("%d", query.After))
	}

	if query.Before > 0 {
		q.Add("before", fmt.Sprintf("%d", query.Before))
	}

	req.URL.RawQuery = q.Encode()

	var rankings LocationClanRankingPager

	if err == nil {
		_, err = i.c.Do(req, &rankings, path)
	}

	return rankings, err
}
