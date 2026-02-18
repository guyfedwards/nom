package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	miniflux "miniflux.app/v2/client"
)

type MinifluxBackend struct {
	Host   string `yaml:"host"`
	APIKey string `yaml:"api_key"`
}

type FreshRSSBackend struct {
	Host       string `yaml:"host"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	PrefixCats bool   `yaml:"prefixCats"`
}

type Backends struct {
	Miniflux []MinifluxBackend `yaml:"miniflux,omitempty"`
	FreshRSS []FreshRSSBackend `yaml:"freshrss,omitempty"`
}

func (mfb *MinifluxBackend) GetFeeds() ([]Feed, error) {
	mf := miniflux.NewClient(mfb.Host, mfb.APIKey)

	// Fetch all feeds.
	feeds, err := mf.Feeds()
	if err != nil {
		return []Feed{}, err
	}

	var ret []Feed

	for _, f := range feeds {
		ret = append(ret, Feed{URL: f.FeedURL, Tags: []string{f.Category.Title}})
	}

	return ret, nil
}

type FreshRSSResponse struct {
	Subscriptions []FreshRSSFeed `yaml:"subscriptions,omitempty"`
}

type Cat struct {
	Label string `yaml:"label,omitempty"`
}

type FreshRSSFeed struct {
	URL        string `yaml:"url,omitempty"`
	Categories []Cat  `yaml:"categories,omitempty"`
}

func (frss FreshRSSFeed) GetCats() string {
	var ret strings.Builder
	for i, v := range frss.Categories {
		if i != 0 {
			ret.WriteByte(',')
		}
		ret.WriteString(v.Label)
	}
	return ret.String()
}

func (frp *FreshRSSBackend) GetFeeds() ([]Feed, error) {
	resp, err := http.Get(fmt.Sprintf("%v/api/greader.php/accounts/ClientLogin?Email=%v&Passwd=%v", frp.Host, frp.User, frp.Password))
	if err != nil {
		return []Feed{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return []Feed{}, fmt.Errorf("could not login to freshrss, statusCode: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []Feed{}, err
	}

	// response is text containing the authorization pair
	lines := strings.Split(string(body), "\n")
	kv := strings.Split(lines[0], "=")

	url := fmt.Sprintf("%v/api/greader.php/reader/api/0/subscription/list?output=json", frp.Host)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []Feed{}, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("GoogleLogin auth=%v", kv[1]))

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return []Feed{}, err
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return []Feed{}, err
	}

	var b FreshRSSResponse
	err = json.Unmarshal(body, &b)
	if err != nil {
		return []Feed{}, err
	}

	var ret []Feed

	for _, f := range b.Subscriptions {
		name := ""
		if frp.PrefixCats {
			name = f.GetCats()
		}

		ret = append(ret, Feed{URL: f.URL, Name: name})
	}

	return ret, nil
}
