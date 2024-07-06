package commands

import (
	"fmt"
	"sync"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/rss"
	"github.com/guyfedwards/nom/v2/internal/store"
)

func (c Commands) CleanFeeds() error {
	urls, err := c.store.GetAllFeedURLs()
	if err != nil {
		return fmt.Errorf("[commands.go]: %w", err)
	}

	var urlsToRemove []string

	for _, u := range urls {
		inFeeds := false
		for _, f := range c.config.Feeds {
			if f.URL == u {
				inFeeds = true
			}
		}

		if !inFeeds {
			urlsToRemove = append(urlsToRemove, u)
		}
	}

	for _, url := range urlsToRemove {
		err := c.store.DeleteByFeedURL(url, false)
		if err != nil {
			return fmt.Errorf("[commands.go]: %w", err)
		}
	}

	return nil
}

func (c Commands) GetAllFeeds() ([]store.Item, error) {
	err := c.CleanFeeds()
	if err != nil {
		return []store.Item{}, fmt.Errorf("[commands.go] GetAllFeeds: %w", err)
	}

	is, err := c.store.GetAllItems()
	if err != nil {
		return []store.Item{}, fmt.Errorf("commands.go: GetAllFeeds %w", err)
	}

	if c.config.ShowFavourites {
		is = onlyFavourites(is)
	} else if c.config.ShowRead {
		is = showRead(is)
	} else {
		is = defaultView(is)
	}

	// add FeedName from config for custom names
	for i := 0; i < len(is); i++ {
		for _, f := range c.config.Feeds {
			if f.URL == is[i].FeedURL {
				is[i].FeedName = f.Name
			}
		}
	}

	return is, nil
}

func onlyFavourites(items []store.Item) (is []store.Item) {
	for _, v := range items {
		if v.Favourite {
			is = append(is, v)
		}
	}

	return is
}

// currently showRead shows all items
func showRead(items []store.Item) (is []store.Item) {
	return items
}

func defaultView(items []store.Item) (is []store.Item) {
	for _, v := range items {
		if !v.Read() {
			is = append(is, v)
		}
	}

	return is
}

func fetchFeed(ch chan FetchResultError, wg *sync.WaitGroup, feed config.Feed, httpOpts *config.HTTPOptions, version string) {
	defer wg.Done()

	r, err := rss.Fetch(feed, httpOpts, version)

	if err != nil {
		ch <- FetchResultError{res: rss.RSS{}, err: err, url: feed.URL}
		return
	}

	ch <- FetchResultError{res: r, err: nil, url: feed.URL}
}
