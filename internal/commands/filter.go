package commands

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/sahilm/fuzzy"

	"github.com/guyfedwards/nom/v2/internal/config"
)

type FilterTerm struct {
	Title     string
	FeedNames []string
	Tags      []string
}

// Struct to aid in filtering items into ranks for BubbleTea
type Filterer struct {
	FeedNames []string
	Tags      []string
	Term      FilterTerm
	Config    config.Config
}

// Generalized function for filtering over a list of options (Used for filtering by feed name and tags)
func (f *Filterer) FilterAgainstStrings(filterValues []string, targetFilterValues []string) fuzzy.Matches {
	// find matching feeds and keep the best matching one in case there are multiple
	ranksGrouped := map[int]fuzzy.Match{}
	for _, feedName := range filterValues {
		matches := fuzzy.Find(feedName, targetFilterValues)
		for _, m := range matches {
			prevMatch, ok := ranksGrouped[m.Index]
			if !ok {
				ranksGrouped[m.Index] = m
			} else {
				if prevMatch.Score < m.Score {
					ranksGrouped[m.Index] = m
				}
			}
		}
	}

	var ranks fuzzy.Matches
	for _, m := range ranksGrouped {
		ranks = append(ranks, m)
	}

	// keep the same order as the input
	// this keeps the same order of items in the UI and prevents the items from being shuffled
	slices.SortStableFunc(ranks, func(left fuzzy.Match, right fuzzy.Match) int {
		return right.Index - left.Index
	})

	return ranks
}

// Breaks what's returned from TUIItem.FilterValue() into a TUIItem.
func (f *Filterer) GetItem(filterValue string) TUIItem {
	splits := strings.Split(filterValue, "||")

	return TUIItem{
		Title:    splits[0],
		FeedName: strings.ToLower(splits[1]),
		Tags:     splits[2:],
	}
}

// Extracts `tag:.*` from the stored f.Term.Title
func (f *Filterer) ExtractFiltersFor(tags ...string) []string {
	var extractedTags []string
	done := false
	for !done {
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
	var targetTags []string

	for _, target := range targets {
		i := f.GetItem(target)
		title := i.Title
		if f.Config.Filtering.DefaultIncludeFeedName {
			title = strings.Join([]string{i.FeedName, i.Title}, " ")
		}
		targetTitles = append(targetTitles, title)
		targetFeedNames = append(targetFeedNames, i.FeedName)
		targetTags = append(targetTags, strings.Join(i.Tags, " "))
	}

	var ranks fuzzy.Matches
	if len(f.FeedNames) > 0 {
		ranks = f.FilterAgainstStrings(f.FeedNames, targetFeedNames)
	} else if len(f.Tags) > 0 {
		ranks = f.FilterAgainstStrings(f.Tags, targetTags)
	} else {
		ranks = fuzzy.Find(f.Term.Title, targetTitles)
	}

	sort.Stable(ranks)

	return ranks
}

func NewFilterer(term string, config config.Config) Filterer {
	f := Filterer{
		Config: config,
		Term: FilterTerm{
			Title: term,
		},
	}

	f.FeedNames = f.ExtractFiltersFor("feedname", "feed", "f")
	f.Tags = f.ExtractFiltersFor("tag", "t")

	return f
}

func CustomFilter(config config.Config) list.FilterFunc {
	return func(term string, targets []string) []list.Rank {
		filterer := NewFilterer(term, config)

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
}
