package rss

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/guyfedwards/nom/v2/internal/config"
)

type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	GUID        string    `xml:"guid"`
	Description string    `xml:"description"`
	Author      string    `xml:"author"`
	Categories  []string  `xml:"categories"`
	Content     string    `xml:"encoded"`
	PubDate     time.Time `xml:"pubDate"`
	FeedName    string
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type RSS struct {
	Channel Channel `xml:"channel"`
}

func Fetch(f config.Feed, httpOpts *config.HTTPOptions, version string) (RSS, error) {
	fp := gofeed.NewParser()

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if httpOpts != nil {
		if version, err := config.TLSVersion(httpOpts.MinTLSVersion); err == nil {
			tr.TLSClientConfig = &tls.Config{
				MinVersion: version,
			}
		}
	}

	fp.Client = &http.Client{
		Transport: tr,
	}

	fp.UserAgent = fmt.Sprintf("nom/%s", version)

	feed, err := fp.ParseURL(f.URL)
	if err != nil {
		return RSS{}, fmt.Errorf("rss.Fetch: %w", err)
	}

	rss := feedToRSS(f, feed)

	return rss, nil
}

func feedToRSS(f config.Feed, feed *gofeed.Feed) RSS {
	items := make([]Item, 0)
	for _, it := range feed.Items {
		ni := Item{
			Title: it.Title,
			Link:  it.Link,
			GUID:  it.GUID,
		}

		if it.Description != "" {
			ni.Description = it.Description
		}

		if it.Author != nil {
			if it.Author.Name != "" {
				ni.Author = it.Author.Name
			}
		}

		if it.Content == "" {
			// If there's no content (as is the case for YouTube RSS items), fallback
			// to the link.
			ni.Content = it.Description
		} else {
			ni.Content = it.Content
		}

		ni.Categories = it.Categories

		// PublishedParsed will be nil if parsing failed
		if it.PublishedParsed != nil {
			ni.PubDate = *it.PublishedParsed
		}

		ni.FeedName = f.Name

		items = append(items, ni)
	}

	rss := RSS{}
	rss.Channel = Channel{
		Title:       feed.Title,
		Link:        feed.Link,
		Description: feed.Description,
		Items:       items,
	}

	return rss
}
