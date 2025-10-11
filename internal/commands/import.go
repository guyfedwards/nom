package commands

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
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
	XMLUrl   *url.URL    `xml:"xmlUrl,attr,omitempty"`
}

func (o *Outline) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	o.XMLName = start.Name
	// Handle custom unmarshal logic for the `Type` and `XMLUrl`
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "type":
			switch attr.Value {
			case RssOutlineType:
				o.Type = OutlineType(attr.Value)
			default:
				return errors.New("Outline.UnmarshalXML: invalid outline type. got " + attr.Value)
			}
		case "xmlUrl":
			URL, err := url.Parse(attr.Value)
			if err != nil {
				return fmt.Errorf("Outline.UnmarshalXML: invalid URL for `xmlUrl`: %w", err)
			}
			o.XMLUrl = URL
		case "text":
			o.Text = attr.Value
		case "title":
			o.Title = attr.Value
		}
	}
	// Recursively decode child outlines
	for {
		token, err := d.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("Outline.UnmarshalXML: unable to parse xml: %w", err)
		}
		switch tt := token.(type) {
		case xml.StartElement:
			var child Outline
			d.DecodeElement(&child, &tt)
			o.Outlines = append(o.Outlines, child)
		}
	}
	return nil
}

func parseOPML(text string) (*OPML, error) {
	var opml OPML
	err := xml.Unmarshal([]byte(text), &opml)
	if err != nil {
		return nil, fmt.Errorf("parseOPML: unable to unmarshal OPML data: %w", err)
	}

	return &opml, nil
}
