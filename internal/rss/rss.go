package rss

import (
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/guyfedwards/nom/internal/config"
)

type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
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

func Fetch(f config.Feed) (RSS, error) {
	fp := gofeed.NewParser()

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
			ni.Content = it.Link
		} else {
			ni.Content = it.Content
		}

		ni.Categories = it.Categories

		pt, err := time.Parse(time.RFC1123Z, it.Published)
		if err != nil {
			// if there is a parsing error, fill with zero-date for now
			pt = time.Time{}
		}

		ni.PubDate = pt
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
