package commands

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/sahilm/fuzzy"
)

// Struct to aid in filtering items into ranks for BubbleTea
type Filterer struct {
	FeedNames []string
	Term      struct {
		Title     string
		FeedNames []string
	}
}

// Breaks what's returned from TUIItem.FilterValue() into a TUIItem.
func (f *Filterer) GetItem(filterValue string) TUIItem {
	var i TUIItem

	splits := strings.Split(filterValue, "||")

	i.Title = splits[0]
	i.FeedName = strings.ToLower(splits[1])

	return i
}

// Extracts `tag:.*` from the stored f.Term.Title
func (f *Filterer) ExtractFiltersFor(tags ...string) []string {
	var extractedTags []string
	done := false
	for done == false {
		// `complete` matches 3 potential capture groups after tags, in which
		// `[^"]` matches a character that isn't a `"`, `[^']` that isn't a `'`,
		// etc. If it's no quotes, you can also do `feed:with\ spaces`
		// `incomplete` matches unfinished quoted tags and removes them from the
		// search. The order of the capture groups MATTERS.
		// In both examples, the %s section matches all potential tag aliases
		// passed in for one tag.
		complete := regexp.MustCompile(fmt.Sprintf(`(%s):("([^"]+)"|'([^']+)'|(([^\\ ]|\\ )+))`, strings.Join(tags, "|")))
		incomplete := regexp.MustCompile(fmt.Sprintf(`(%s):("[^"]*|'[^']*)`, strings.Join(tags, "|")))

		matches := complete.FindStringSubmatch(f.Term.Title)

		match := ""
		if matches != nil {
			// double quotes
			if matches[3] != "" {
				match = matches[3]
				// single quotes
			} else if matches[4] != "" {
				match = matches[4]
				// no quotes
			} else if matches[5] != "" {
				match = strings.ReplaceAll(matches[5], `\ `, " ")
			}
			f.Term.Title = strings.Replace(f.Term.Title, matches[0], "", 1)
		} else {
			// fallback to regular matching without filter
			matches = incomplete.FindStringSubmatch(f.Term.Title)
			if matches != nil {
				f.Term.Title = strings.Replace(f.Term.Title, matches[0], "", 1)
			}
			done = true
		}

		if match != "" {
			extractedTags = append(extractedTags, strings.ToLower(match))
		}
	}
	if f.Term.Title == "" {
		f.Term.Title = " "
	}

	return extractedTags
}

// Runs all filters
func (f *Filterer) Filter(targets []string) []fuzzy.Match {
	var targetTitles []string
	var targetFeedNames []string

	for _, target := range targets {
		i := f.GetItem(target)
		targetTitles = append(targetTitles, i.Title)
		targetFeedNames = append(targetFeedNames, i.FeedName)
	}

	var ranks fuzzy.Matches
	if len(f.FeedNames) == 0 {
		ranks = fuzzy.Find(f.Term.Title, targetTitles)
	} else {
		for _, feedName := range f.FeedNames {
			ranks = append(ranks, fuzzy.Find(feedName, targetFeedNames)...)
		}
	}

	sort.Stable(ranks)

	return ranks
}

func NewFilterer(term string) Filterer {
	var f Filterer

	f.Term.Title = term
	f.FeedNames = f.ExtractFiltersFor("feedname", "feed", "f")

	return f
}

func CustomFilter(term string, targets []string) []list.Rank {
	filterer := NewFilterer(term)

	ranks := filterer.Filter(targets)

	result := make([]list.Rank, len(ranks))
	for i, rank := range ranks {
		result[i] = list.Rank{
			Index:          rank.Index,
			MatchedIndexes: rank.MatchedIndexes,
		}
	}

	return result
}
