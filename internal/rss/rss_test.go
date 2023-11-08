package rss

import (
	"bytes"
	"os"
	"testing"

	"github.com/mmcdole/gofeed"

	"github.com/guyfedwards/nom/internal/config"

	"github.com/guyfedwards/nom/internal/test"
)

const dropboxFixture = "../test/data/dropbox_fixture.rss"

func getFixtureAsFeed(t *testing.T) *gofeed.Feed {
	f, err := os.ReadFile(dropboxFixture)
	if err != nil {
		t.Logf("error getting fixture: %e", err)
		t.Fail()
		return nil
	}

	p := gofeed.NewParser()
	feed, err := p.Parse(bytes.NewReader(f))
	if err != nil {
		t.Logf("error getting fixture: %e", err)
		t.Fail()
		return nil
	}

	return feed
}

func TestFeedToRss(t *testing.T) {
	fd := getFixtureAsFeed(t)

	r := feedToRSS(config.Feed{Name: "bigup"}, fd)

	test.Equal(t, "We are venom", r.Channel.Title, "bad channel title")
	test.Equal(t, "foobar.com", r.Channel.Link, "bad channel link")
	test.Equal(t, "Need brains", r.Channel.Description, "bad description")

	test.Equal(t, 10, len(r.Channel.Items), "missing items")

	test.Equal(t, "Using OAuth 2.0 with offline access", r.Channel.Items[0].Title, "bad title")
	test.Equal(t, "https://dropbox.tech/developers/using-oauth", r.Channel.Items[0].Link, "bad link")
	test.Equal(t, "OAuth flow", r.Channel.Items[0].Categories[0], "bad category")
	test.Equal(t, "Authorization", r.Channel.Items[0].Categories[1], "bad category 2 ")
	test.Equal(t, "Learn how to use the Dropbox OAuth", r.Channel.Items[0].Description, "bad item description")
	test.Equal(t, "Dropbox Platform Team", r.Channel.Items[0].Author, "bad item author")
	test.Equal(t, "bigup", r.Channel.Items[0].FeedName, "bad feedname")
	test.Equal(t, 1666161000, r.Channel.Items[0].PubDate.Unix(), "bad feedname")

}
