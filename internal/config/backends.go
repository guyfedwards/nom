package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	miniflux "miniflux.app/client"
)

func getMinifluxFeeds(config *MinifluxBackend) ([]Feed, error) {
	mf := miniflux.New(config.Host, config.APIKey)

	// Fetch all feeds.
	feeds, err := mf.Feeds()
	if err != nil {
		return []Feed{}, err
	}

	var ret []Feed

	for _, f := range feeds {
		ret = append(ret, Feed{URL: f.FeedURL})
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
	ret := ""
	for i, v := range frss.Categories {
		if i != 0 {
			ret += ","
		}
		ret += v.Label
	}
	return ret
}

func getFreshRSSFeeds(config *FreshRSSBackend) ([]Feed, error) {
	resp, err := http.Get(fmt.Sprintf("%v/api/greader.php/accounts/ClientLogin?Email=%v&Passwd=%v", config.Host, config.User, config.Password))
	if err != nil {
		return []Feed{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return []Feed{}, fmt.Errorf("could not login to freshrss, statusCode: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []Feed{}, err
	}

	// response is text containing the authorization pair
	lines := strings.Split(string(body), "\n")
	kv := strings.Split(lines[0], "=")

	url := fmt.Sprintf("%v/api/greader.php/reader/api/0/subscription/list?output=json", config.Host)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []Feed{}, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("GoogleLogin auth=%v", kv[1]))

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return []Feed{}, err
	}

	body, err = ioutil.ReadAll(resp.Body)
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
		if config.PrefixCats {
			name = f.GetCats()
		}
		ret = append(ret, Feed{URL: f.URL, Name: name})
	}

	return ret, nil
}
