package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/list"

	"github.com/guyfedwards/nom/internal/cache"
	"github.com/guyfedwards/nom/internal/config"
	"github.com/guyfedwards/nom/internal/rss"
)

type Commands struct {
	config config.Config
	cache  cache.Cache
}

func New(config config.Config, cache cache.Cache) Commands {
	return Commands{config, cache}
}

func (c Commands) TUI() error {
	rsss, err := c.fetchAllFeeds(c.config.Feeds)
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	var items []list.Item

	for _, r := range rsss {
		for _, item := range r.Channel.Items {
			items = append(items, RSSToItem(item))
		}
	}

	if err := Render(items, c); err != nil {
		return fmt.Errorf("commands.TUI: %w", err)
	}

	return nil
}

func (c Commands) List(numResults int, cache bool) error {
	rsss, err := c.fetchAllFeeds(c.config.Feeds)
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	output := ""

	for _, r := range rsss {
		for _, item := range r.Channel.Items {
			output += fmt.Sprintf("%s \n  - %s\n", item.Title, item.Link)
		}
	}

	if c.config.Pager == "false" {
		fmt.Println(output)
		return nil
	}

	return outputToPager(output)
}

func (c Commands) Add(url string) error {
	err := c.config.AddFeed(config.Feed{URL: url})
	if err != nil {
		return fmt.Errorf("commands Add: %w", err)
	}

	return nil
}

type FetchResultError struct {
	res rss.RSS
	err error
	url string
}

func (c Commands) fetchAllFeeds(feeds []config.Feed) ([]rss.RSS, error) {
	var (
		rsss []rss.RSS
		wg   sync.WaitGroup
	)

	if len(feeds) <= 0 {
		return []rss.RSS{}, fmt.Errorf("no feeds found, add to nom/config.yml")
	}

	ch := make(chan FetchResultError)

	for _, feed := range feeds {
		v, err := c.cache.Read(feed.URL)

		if c.config.NoCache || err == cache.ErrCacheMiss {
			wg.Add(1)
			go fetchFeed(ch, &wg, feed.URL)
		} else if err != nil {
			log.Fatal("error getting cache")
		} else {
			wg.Add(1)
			go func(feed config.Feed) {
				ch <- FetchResultError{res: v, err: nil, url: feed.URL}
				wg.Done()
			}(feed)
		}
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		// TODO: handle error more gracefully per feed and resort to cache
		if result.err != nil {
			close(ch)

			return []rss.RSS{}, fmt.Errorf("commands List: %w", result.err)
		}

		rsss = append(rsss, result.res)

		err := c.cache.Write(result.url, result.res)
		if err != nil {
			log.Fatal("Error writing to cache")
		}
	}

	return rsss, nil
}

func fetchFeed(ch chan FetchResultError, wg *sync.WaitGroup, feedURL string) {
	defer wg.Done()

	r, err := rss.Fetch(feedURL)

	if err != nil {
		ch <- FetchResultError{res: rss.RSS{}, err: err, url: feedURL}
		return
	}

	ch <- FetchResultError{res: r, err: nil, url: feedURL}
}

func (c Commands) FindArticle(substr string) (item rss.Item, err error) {
	rsss, err := c.fetchAllFeeds(c.config.Feeds)
	if err != nil {
		return rss.Item{}, fmt.Errorf("commands.FindArticle: %w", err)
	}

	regex, err := regexp.Compile(strings.ToLower(substr))
	if err != nil {
		return rss.Item{}, fmt.Errorf("commands.FindArticle: regexp: %w", err)
	}

	for _, r := range rsss {
		for _, it := range r.Channel.Items {
			// very basic string matching on title to read an article
			if regex.MatchString(strings.ToLower(it.Title)) {
				item = it
				break
			}
		}
	}

	return item, nil
}

func (c Commands) FindGlamourisedArticle(substr string) (string, error) {
	article, err := c.FindArticle(substr)
	if err != nil {
		return "", fmt.Errorf("commands.FindGlamourisedArticle: %w", err)
	}

	content, err := rss.GlamouriseItem(article)
	if err != nil {
		return "", fmt.Errorf("commands Read: %w", err)
	}

	return content, nil
}

func (c Commands) Read(substrs ...string) error {
	substr := strings.Join(substrs, " ")

	content, err := c.FindGlamourisedArticle(substr)
	if err != nil {
		return fmt.Errorf("commands.Read: %w", err)
	}

	if c.config.Pager == "false" {
		fmt.Println(content)
		return nil
	}

	return outputToPager(content)
}

func outputToPager(content string) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -r"
	}

	pa := strings.Split(pager, " ")
	cmd := exec.Command(pa[0], pa[1:]...)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout

	return cmd.Run()
}
