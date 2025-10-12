package commands

import (
	"os"
	"slices"
	"testing"

	"github.com/guyfedwards/nom/v2/internal/test"
)

const opmlFixture = "../test/data/test.opml"

var folders []string = []string{
	"News",
	"Sports",
	"Leisure",
	"Tech",
}

var feeds map[string]string = map[string]string{
	"Big News Finland":          "http://www.bignewsnetwork.com/?rss=37e8860164ce009a",
	"Euronews":                  "http://feeds.feedburner.com/euronews/en/news/",
	"Reuters Top News":          "http://feeds.reuters.com/reuters/topNews",
	"Yahoo Europe":              "http://rss.news.yahoo.com/rss/europe",
	"CNN Entertainment":         "http://rss.cnn.com/rss/edition_entertainment.rss",
	"E! News":                   "http://uk.eonline.com/syndication/feeds/rssfeeds/topstories.xml",
	"Hollywood Reporter":        "http://feeds.feedburner.com/thr/news",
	"Reuters Entertainment":     "http://feeds.reuters.com/reuters/entertainment",
	"Reuters Music News":        "http://feeds.reuters.com/reuters/musicNews",
	"Yahoo Entertainment":       "http://rss.news.yahoo.com/rss/entertainment",
	"Formula 1":                 "http://www.formula1.com/rss/news/latest.rss",
	"MotoGP":                    "http://rss.crash.net/crash_motogp.xml",
	"N.Y.Times Track And Field": "http://topics.nytimes.com/topics/reference/timestopics/subjects/t/track_and_field/index.html?rss=1",
	"Reuters Sports":            "http://feeds.reuters.com/reuters/sportsNews",
	"Yahoo Sports NHL":          "http://sports.yahoo.com/nhl/rss.xml",
	"Yahoo Sports":              "http://rss.news.yahoo.com/rss/sports",
	"Coding Horror":             "http://feeds.feedburner.com/codinghorror/",
	"Gadget Lab":                "http://www.wired.com/gadgetlab/feed/",
	"Gizmodo":                   "http://gizmodo.com/index.xml",
	"Reuters Technology":        "http://feeds.reuters.com/reuters/technologyNews",
}

func TestOMPLParser(t *testing.T) {
	bytes, err := os.ReadFile(opmlFixture)
	if err != nil {
		t.Fatalf("unable to read opml fixture file: %s", err)
	}
	result, err := parseOPML(bytes)
	test.HandleError(t, err)
	test.Equal(t, "1.0", result.Version, "incorrect opml version")
	test.Equal(t, "Sample OPML file for RSSReader", result.Head.Title, "incorrect document title")
	test.Equal(t, len(result.Body.Outlines), 4, "missing outline folders")
	for _, outline := range result.Body.Outlines {
		if !slices.Contains(folders, outline.Title) || !slices.Contains(folders, outline.Text) {
			t.Fatalf("invalid folder outline: %+v", outline)
		}
		switch outline.Title {
		case "News":
			test.Equal(t, len(outline.Outlines), 4, "missing outlines for News")
		case "Sports":
			test.Equal(t, len(outline.Outlines), 6, "missing outlines for Sports")
		case "Leisure":
			test.Equal(t, len(outline.Outlines), 6, "missing outlines for Leisure")
		case "Tech":
			test.Equal(t, len(outline.Outlines), 4, "missing outlines for Tech")

		}
		for _, child := range outline.Outlines {
			test.Equal(t, RssOutlineType, child.Type, "invalid outline type: "+string(child.Type))
			test.Equal(t, feeds[child.Title], child.XMLUrl.String(), "invalid feed")
		}
	}
}
