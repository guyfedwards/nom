package rss

import (
	"fmt"
	"log"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/glamour"
	"github.com/mmcdole/gofeed"
)

type Item struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	Author      string  `xml:"author"`
	Category    string  `xml:"category"`
	Content     string  `xml:"encoded"`
	PubDate     pubDate `xml:"pubDate"`
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

func GlamouriseItem(item Item) (string, error) {
	var mdown string

	mdown += "# " + item.Title
	mdown += "\n"
	mdown += item.Author
	mdown += "\n"
	mdown += item.PubDate.String()
	mdown += "\n\n"
	mdown += htmlToMd(item.Content)

	out, err := glamour.Render(mdown, "dark")
	if err != nil {
		return "", fmt.Errorf("GlamouriseItem: %w", err)
	}

	return out, nil
}

func htmlToMd(html string) string {
	converter := md.NewConverter("", true, nil)

	mdown, err := converter.ConvertString(html)
	if err != nil {
		log.Fatal(err)
	}

	return mdown
}

type pubDate struct {
	time.Time
}

func Fetch(feedURL string) (RSS, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return RSS{}, fmt.Errorf("rss.Fetch: %w", err)
	}
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
		// TODO: support multiple categories
		if len(it.Categories) > 0 {
			ni.Category = it.Categories[0]
		}

		var pd pubDate
		pt, err := time.Parse(time.RFC1123Z, it.Published)
		if err != nil {
			// if there is a parsing error, fill with zero-date for now
			pd = pubDate{time.Time{}}
		} else {
			pd = pubDate{pt}
		}
		ni.PubDate = pd
		items = append(items, ni)
	}
	rss := RSS{}
	rss.Channel = Channel{
		Title:       feed.Title,
		Link:        feed.Link,
		Description: feed.Description,
		Items:       items,
	}
	return rss, nil
}
