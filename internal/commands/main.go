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
	cache  cache.CacheInterface
}

func New(config config.Config, cache cache.CacheInterface) Commands {
	return Commands{config, cache}
}

func getItemsFromRSS(rsss []rss.RSS) []list.Item {
	var items []list.Item

	for _, r := range rsss {
		for _, item := range r.Channel.Items {
			items = append(items, RSSToItem(item))
		}
	}

	return items
}

func (c Commands) TUI() error {
	rsss, err := c.fetchAllFeeds(false)
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	items := getItemsFromRSS(rsss)

	if err := Render(items, c); err != nil {
		return fmt.Errorf("commands.TUI: %w", err)
	}

	return nil
}

func (c Commands) List(numResults int, cache bool) error {
	rsss, err := c.fetchAllFeeds(false)
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

func (c Commands) fetchAllFeeds(noCacheOverride bool) ([]rss.RSS, error) {
	var (
		rsss []rss.RSS
		wg   sync.WaitGroup
	)

	feeds := c.config.GetFeeds()

	if len(feeds) <= 0 {
		return []rss.RSS{}, fmt.Errorf("no feeds found, add to nom/config.yml")
	}

	ch := make(chan FetchResultError)

	for _, feed := range feeds {
		v, err := c.cache.Read(feed.URL)

		if c.config.NoCache || noCacheOverride || err == cache.ErrCacheMiss {
			wg.Add(1)

			go fetchFeed(ch, &wg, feed)
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

func fetchFeed(ch chan FetchResultError, wg *sync.WaitGroup, feed config.Feed) {
	defer wg.Done()

	r, err := rss.Fetch(feed)

	if err != nil {
		ch <- FetchResultError{res: rss.RSS{}, err: err, url: feed.URL}
		return
	}

	ch <- FetchResultError{res: r, err: nil, url: feed.URL}
}

func (c Commands) FindArticle(substr string) (item rss.Item, err error) {
	rsss, err := c.fetchAllFeeds(false)
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
