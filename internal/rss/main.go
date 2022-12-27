package rss

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/glamour"
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

func (pd *pubDate) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var value string

	err := d.DecodeElement(&value, &start)
	if err != nil {
		return fmt.Errorf("rss unmarshalxml: %w", err)
	}

	parse, err := time.Parse(time.RFC1123Z, value)
	if err != nil {
		// if there is a parsing error, fill with zero-date for now
		*pd = pubDate{time.Time{}}
		return nil
	}

	*pd = pubDate{parse}

	return nil
}

func Fetch(feedURL string) (RSS, error) {
	resp, err := http.Get(feedURL)
	if err != nil {
		return RSS{}, fmt.Errorf("rss.Fetch: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return RSS{}, fmt.Errorf("rss.Fetch: %w", err)
	}

	var rss RSS

	err = xml.Unmarshal(body, &rss)
	if err != nil {
		return RSS{}, fmt.Errorf("rss.Fetch: %w", err)
	}

	return rss, nil
}
