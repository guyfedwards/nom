package commands

import (
	"encoding/xml"
	"net/url"
	"os"
)

type OutlineType string

const (
	RssOutlineType = "rss"
)

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head     `xml:"head"`
	Body    Body     `xml:"body"`
}

type Head struct {
	Title string `xml:"title,omitempty"`
}

type Body struct {
	Outlines []Outline `xml:"outline"`
}

type Outline struct {
	XMLName  xml.Name    `xml:"outline"`
	Outlines []Outline   `xml:"outline,omitempty"`
	Text     string      `xml:"text,attr"`
	Title    string      `xml:"title,attr,omitempty"`
	Type     OutlineType `xml:"type,attr,omitempty"`
	XMLUrl   url.URL     `xml:"xmlUrl,attr,omitempty"`
}

func parseOPML(file *os.File) (*OPML, error) {
	var opml OPML
	decoder := xml.NewDecoder(file)
	err := decoder.Decode(&opml)
	if err != nil {
		return nil, err
	}

	return &opml, nil
}
