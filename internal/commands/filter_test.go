package commands

import (
	"testing"

	"github.com/guyfedwards/nom/v2/internal/config"
)

var testItems = []string{
	"Introduction to Golang||tech blog||programming||golang",
	"Python tutorial||dev blog||programming||python",
	"Breaking news||hacker news||world news||important",
	"JavaScript framework||web blog||javascript||frontend",
	"Guide to Rust||the rust blog||programming||rust",
	"Advanced Go topics||tech blog||programming||golang||advanced",
}

func TestFilter_SimpleTextSearch(t *testing.T) {
	cfg := config.Config{
		Filtering: config.FilterConfig{
			DefaultIncludeFeedName: false,
		},
	}

	testCases := []struct {
		name          string
		searchTerm    string
		expectedCount int
		expectedFirst int // index of first expected match
	}{
		{
			name:          "exact match",
			searchTerm:    "golang",
			expectedCount: 1,
			expectedFirst: 0,
		},
		{
			name:          "fuzzy match",
			searchTerm:    "intro",
			expectedCount: 1,
			expectedFirst: 0,
		},
		{
			name:          "multiple matches",
			searchTerm:    "Go",
			expectedCount: 3,
		},
		{
			name:          "no matches",
			searchTerm:    "nonexistent",
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filterer := NewFilterer(tc.searchTerm, cfg)
			matches := filterer.Filter(testItems)

			if len(matches) != tc.expectedCount {
				t.Errorf("expected %d matches, got %d", tc.expectedCount, len(matches))
			}

			// Only check expectedFirst if it's explicitly set (non-zero)
			if tc.expectedFirst > 0 && len(matches) > 0 {
				if matches[0].Index != tc.expectedFirst {
					t.Errorf("expected first match at index %d, got %d", tc.expectedFirst, matches[0].Index)
				}
			} else if tc.expectedFirst == 0 && tc.expectedCount == 1 && len(matches) > 0 {
				// Special case: when expectedFirst is 0 and we expect exactly 1 match
				if matches[0].Index != 0 {
					t.Errorf("expected first match at index 0, got %d", matches[0].Index)
				}
			}
		})
	}
}

func TestFilter_FeedNameSearch(t *testing.T) {
	cfg := config.Config{
		Filtering: config.FilterConfig{
			DefaultIncludeFeedName: false,
		},
	}

	testCases := []struct {
		name          string
		searchTerm    string
		expectedCount int
		expectedIndex int
	}{
		{
			name:          "feed prefix",
			searchTerm:    "feed:tech",
			expectedCount: 2,
			expectedIndex: 0,
		},
		{
			name:          "feedname prefix",
			searchTerm:    "feedname:dev",
			expectedCount: 1,
			expectedIndex: 1,
		},
		{
			name:          "f prefix (short form)",
			searchTerm:    "f:hacker",
			expectedCount: 1,
			expectedIndex: 2,
		},
		{
			name:          "feed with double quotes",
			searchTerm:    `feed:"web blog"`,
			expectedCount: 1,
			expectedIndex: 3,
		},
		{
			name:          "feed with single quotes",
			searchTerm:    `feed:'hacker news'`,
			expectedCount: 1,
			expectedIndex: 2,
		},
		{
			name:          "no matches",
			searchTerm:    "feed:nonexistent",
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filterer := NewFilterer(tc.searchTerm, cfg)
			matches := filterer.Filter(testItems)

			if len(matches) != tc.expectedCount {
				t.Errorf("expected %d matches, got %d", tc.expectedCount, len(matches))
			}

			if tc.expectedCount > 0 && len(matches) > 0 {
				if matches[0].Index != tc.expectedIndex {
					t.Errorf("expected match at index %d, got %d", tc.expectedIndex, matches[0].Index)
				}
			}
		})
	}
}

func TestFilter_TagSearch(t *testing.T) {
	cfg := config.Config{
		Filtering: config.FilterConfig{
			DefaultIncludeFeedName: false,
		},
	}

	testCases := []struct {
		name          string
		searchTerm    string
		expectedCount int
		expectedIndex int
	}{
		{
			name:          "tag prefix",
			searchTerm:    "tag:programming",
			expectedCount: 4,
		},
		{
			name:          "t prefix (short form)",
			searchTerm:    "t:golang",
			expectedCount: 2,
			expectedIndex: 0,
		},
		{
			name:          "tag with double quotes",
			searchTerm:    `tag:"world news"`,
			expectedCount: 1,
			expectedIndex: 2,
		},
		{
			name:          "tag with single quotes",
			searchTerm:    `tag:'world news'`,
			expectedCount: 1,
			expectedIndex: 2,
		},
		{
			name:          "no matches",
			searchTerm:    "tag:backend",
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filterer := NewFilterer(tc.searchTerm, cfg)
			matches := filterer.Filter(testItems)

			if len(matches) != tc.expectedCount {
				t.Errorf("expected %d matches, got %d", tc.expectedCount, len(matches))
			}

			if tc.expectedCount > 0 && len(matches) > 0 && tc.expectedIndex > 0 {
				if matches[0].Index != tc.expectedIndex {
					t.Errorf("expected first match at index %d, got %d", tc.expectedIndex, matches[0].Index)
				}
			}
		})
	}
}

func TestFilter_QuoteHandling(t *testing.T) {
	cfg := config.Config{
		Filtering: config.FilterConfig{
			DefaultIncludeFeedName: false,
		},
	}

	testCases := []struct {
		name          string
		searchTerm    string
		expectedCount int
		expectedIndex int
	}{
		{
			name:          "double quotes with spaces",
			searchTerm:    `feed:"tech blog"`,
			expectedCount: 2,
			expectedIndex: 0,
		},
		{
			name:          "single quotes with spaces",
			searchTerm:    `feed:'hacker news'`,
			expectedCount: 1,
			expectedIndex: 2,
		},
		{
			name:          "backslash escaped spaces",
			searchTerm:    `feed:the\ rust\ blog`,
			expectedCount: 1,
			expectedIndex: 4,
		},
		{
			name:          "no quotes for single word",
			searchTerm:    "feed:hacker",
			expectedCount: 1,
			expectedIndex: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filterer := NewFilterer(tc.searchTerm, cfg)
			matches := filterer.Filter(testItems)

			if len(matches) != tc.expectedCount {
				t.Errorf("expected %d matches, got %d", tc.expectedCount, len(matches))
			}

			if tc.expectedCount > 0 && len(matches) > 0 {
				if matches[0].Index != tc.expectedIndex {
					t.Errorf("expected match at index %d, got %d", tc.expectedIndex, matches[0].Index)
				}
			}
		})
	}
}

func TestFilter_DefaultIncludeFeedName(t *testing.T) {
	cfgWithFeedName := config.Config{
		Filtering: config.FilterConfig{
			DefaultIncludeFeedName: true,
		},
	}

	cfgWithoutFeedName := config.Config{
		Filtering: config.FilterConfig{
			DefaultIncludeFeedName: false,
		},
	}

	t.Run("search feed name without prefix when enabled", func(t *testing.T) {
		filterer := NewFilterer("hacker", cfgWithFeedName)
		matches := filterer.Filter(testItems)

		if len(matches) != 1 {
			t.Errorf("expected 1 match when DefaultIncludeFeedName is true, got %d", len(matches))
		}

		if len(matches) > 0 && matches[0].Index != 2 {
			t.Errorf("expected match at index 2, got %d", matches[0].Index)
		}
	})

	t.Run("don't search feed name without prefix when disabled", func(t *testing.T) {
		filterer := NewFilterer("hacker", cfgWithoutFeedName)
		matches := filterer.Filter(testItems)

		if len(matches) != 0 {
			t.Errorf("expected 0 matches when DefaultIncludeFeedName is false, got %d", len(matches))
		}
	})
}
