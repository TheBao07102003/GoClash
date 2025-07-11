package clash

import (
	"fmt"
	"time"
)

type ClanQuery struct {
	PagedQuery
	LocationId int
	MinScore   int
	MinMembers int
	MaxMembers int
	Name       string
}

type Clan struct {
	Tag               string       `json:"tag"`
	Name              string       `json:"name"`
	Type              string       `json:"type"`
	Description       string       `json:"description"`
	BadgeId           int          `json:"badgeId"`
	ClanScore         int          `json:"clanScore"`
	ClanWarTrophies   int          `json:"clanWarTrophies"`
	Location          Location     `json:"location"`
	RequiredTrophies  int          `json:"requiredTrophies"`
	DonationsPerWeek  int          `json:"donationsPerWeek"`
	ClanChestStatus   string       `json:"clanChestStatus"`
	ClanChestLevel    int          `json:"clanChestLevel"`
	ClanChestMaxLevel int          `json:"clanChestMaxLevel"`
	Members           int          `json:"members"`
	MemberList        []ClanMember `json:"memberList"`
}

type ClanPager struct {
	Items  []Clan `json:"items"`
	Paging Paging `json:"paging"`
}

type MemberPager struct {
	Items  []ClanMember `json:"items"`
	Paging Paging       `json:"paging"`
}

type WarParticipant struct {
	Tag          string `json:"tag"`
	Name         string `json:"name"`
	Fame         int    `json:"fame"`
	RepairPoints int    `json:"repairPoints"`
	BoatAttacks  int    `json:"boatAttacks"`
	DecksUsed    int    `json:"decksUsed"`
}

type WarClanDetails struct {
	Tag           string           `json:"tag"`
	Name          string           `json:"name"`
	BadgeId       int              `json:"badgeId"`
	Fame          int              `json:"fame"`
	RepairPoints  int              `json:"repairPoints"`
	Participants  []WarParticipant `json:"participants"`
	ClanScore     int              `json:"clanScore"`
	RawFinishTime string           `json:"finishTime"`
}

func (w *WarClanDetails) FinishTime() time.Time {
	parsed, _ := time.Parse(TimeLayout, w.RawFinishTime)
	return parsed
}

type WarStanding struct {
	Rank         int            `json:"rank"`
	TrophyChange int            `json:"trophyChange"`
	Clan         WarClanDetails `json:"clan"`
}

type War struct {
	Standings      []WarStanding    `json:"standings"`
	SeasonId       int              `json:"seasonId"`
	RawCreatedDate string           `json:"createdDate"`
	Participants   []WarParticipant `json:"participants"`
	SectionIndex   int              `json:"sectionIndex"`
}

func (w *War) CreatedDate() time.Time {
	parsed, _ := time.Parse(TimeLayout, w.RawCreatedDate)
	return parsed
}

type WarLogPager struct {
	Items  []War  `json:"items"`
	Paging Paging `json:"paging"`
}

type CurrentWar struct {
	State        string           `json:"state"`
	Clan         WarClanDetails   `json:"clan"`
	Clans        []WarClanDetails `json:"clans"`
	Participants []WarParticipant `json:"participants"`
	SectionIndex int              `json:"sectionIndex"`
}

type ClanMember struct {
	Tag               string `json:"tag"`
	Name              string `json:"name"`
	Role              string `json:"role"`
	RawLastSeen       string `json:"lastSeen"`
	ExpLevel          int    `json:"expLevel"`
	Trophies          int    `json:"trophies"`
	Arena             Arena  `json:"arena"`
	ClanRank          int    `json:"clanRank"`
	PreviousClanRank  int    `json:"previousClanRank"`
	Donations         int    `json:"donations"`
	DonationsReceived int    `json:"donationsReceived"`
	clanChestPoints   int    `json:"clanChestPoints"`
}

func (c *ClanMember) LastSeen() time.Time {
	parsed, _ := time.Parse(TimeLayout, c.RawLastSeen)
	return parsed
}

type ClansService struct {
	c *Client
}

type ClanService struct {
	c   *Client
	tag string
}

func (c *Client) Clans() *ClansService {
	return &ClansService{c}
}

func (c *Client) Clan(tag string) *ClanService {
	return &ClanService{c, tag}
}

// Get information about a single clan by clan tag.
// Clan tags can be found using clan search operation.
func (i *ClanService) Get() (Clan, error) {
	path := "/v1/clans/%s"
	url := fmt.Sprintf(path, NormaliseTag(i.tag))
	req, err := i.c.NewRequest("GET", url, nil)
	var clan Clan

	if err == nil {
		_, err = i.c.Do(req, &clan, path)
	}

	return clan, err
}

// Retrieve information about clan's current clan war
func (i *ClanService) CurrentWar() (CurrentWar, error) {
	path := "/v1/clans/%s/currentriverrace"
	url := fmt.Sprintf(path, NormaliseTag(i.tag))
	req, err := i.c.NewRequest("GET", url, nil)
	var war CurrentWar

	if err == nil {
		_, err = i.c.Do(req, &war, path)
	}

	return war, err
}

// Retrieve clan's clan war log
func (i *ClanService) WarLog() (WarLogPager, error) {
	path := "/v1/clans/%s/riverracelog"
	url := fmt.Sprintf(path, NormaliseTag(i.tag))
	req, err := i.c.NewRequest("GET", url, nil)
	var warLog WarLogPager

	if err == nil {
		_, err = i.c.Do(req, &warLog, path)
	}

	return warLog, err
}

// List clan members
func (i *ClanService) Members() (MemberPager, error) {
	path := "/v1/clans/%s/members"
	url := fmt.Sprintf(path, NormaliseTag(i.tag))
	req, err := i.c.NewRequest("GET", url, nil)
	var members MemberPager

	if err == nil {
		_, err = i.c.Do(req, &members, path)
	}

	return members, err
}

// Search all clans by name and/or filtering the results using various criteria.
// At least one filtering criteria must be defined and if name is used
// as part of search, it is required to be at least three characters long.
func (i *ClansService) Search(query *ClanQuery) (ClanPager, error) {
	path := "/v1/clans"
	req, err := i.c.NewRequest("GET", path, nil)
	q := req.URL.Query()

	if query.LocationId > 0 {
		q.Add("locationId", fmt.Sprintf("%d", query.LocationId))
	}

	if query.MinScore > 0 {
		q.Add("minScore", fmt.Sprintf("%d", query.MinScore))
	}

	// Yes, what you're reading is correct, minMembers needs to be >= 2
	if query.MinMembers >= 2 {
		q.Add("minMembers", fmt.Sprintf("%d", query.MinMembers))
	}

	// maxMembers cannot be zero
	if query.MaxMembers >= 1 && query.MaxMembers <= 50 {
		q.Add("maxMembers", fmt.Sprintf("%d", query.MaxMembers))
	}

	if len(query.Name) >= 3 {
		q.Add("name", query.Name)
	}

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

	var clans ClanPager

	if err == nil {
		_, err = i.c.Do(req, &clans, path)
	}

	return clans, err
}
