package rss

import (
	"bytes"
	"os"
	"testing"

	"github.com/mmcdole/gofeed"

	"github.com/guyfedwards/nom/v2/internal/config"

	"github.com/guyfedwards/nom/v2/internal/test"
)

const dropboxFixture = "../test/data/dropbox_fixture.rss"
const badPubDateFixture = "../test/data/bad_pub_date.rss"

func getFixtureAsFeed(path string) (*gofeed.Feed, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	p := gofeed.NewParser()
	feed, err := p.Parse(bytes.NewReader(f))
	if err != nil {
		return nil, err
	}

	return feed, nil
}

func TestFeedToRss(t *testing.T) {
	fd, err := getFixtureAsFeed(dropboxFixture)
	if err != nil {
		t.Logf("error getting fixture: %e", err)
		t.Fail()
	}

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

func TestFeedPubDateBadParse(t *testing.T) {
	fd, err := getFixtureAsFeed(badPubDateFixture)
	if err != nil {
		t.Logf("error getting fixture: %e", err)
		t.Fail()
	}

	r := feedToRSS(config.Feed{}, fd)
	test.Equal(t, "0001-01-01 00:00:00 +0000 UTC", r.Channel.Items[0].PubDate.String(), "dates don't match")
}
