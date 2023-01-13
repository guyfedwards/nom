package cache

import (
	"errors"
	"testing"

	"github.com/guyfedwards/nom/internal/rss"
)

func TestReadReturnError(t *testing.T) {
	c := NewFileCache("../test/data", 0)

	_, err := c.Read("foo")
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("wrong error type")
	}
}

func TestWrite(t *testing.T) {
	c := NewFileCache("../test/data/", 0)

	err := c.Write("keytomyheart", rss.RSS{
		Channel: rss.Channel{
			Title:       "thisissparta",
			Link:        "keytomyheart",
			Description: "so much content",
			Items:       []rss.Item{},
		},
	})

	if err != nil {
		t.Fail()
	}
}

func TestRead(t *testing.T) {
	c := NewFileCache("../test/data/", 0)

	_ = c.Write("cashin", rss.RSS{
		Channel: rss.Channel{
			Title:       "cashout",
			Link:        "cashin",
			Description: "21 21 21 21",
			Items:       []rss.Item{},
		},
	})

	v, err := c.Read("cashin")

	if err != nil {
		t.Fail()
	}

	if v.Channel.Title != "cashout" || v.Channel.Link != "cashin" {
		t.Fail()
	}
}
