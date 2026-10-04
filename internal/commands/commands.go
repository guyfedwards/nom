package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/thebanri/limoni"
	"gopkg.in/yaml.v3"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/rss"
	"github.com/guyfedwards/nom/v2/internal/store"
)

type Commands struct {
	config *config.Config
	store  store.Store
}

func New(config *config.Config, store store.Store) *Commands {
	return &Commands{config, store}
}

func convertItems(its []store.Item) []TUIItem {
	var items []TUIItem

	for _, item := range its {
		items = append(items, ItemToTUIItem(item))
	}

	return items
}

func IsWSL() bool {
	out, err := exec.Command("uname", "-a").Output()
	if err != nil {
		return false
	}
	// In some cases, uname on wsl outputs microsoft capitalized
	matched, _ := regexp.Match(`microsoft|Microsoft`, out)
	return matched
}

func IsWayland() bool {
	s := os.Getenv("XDG_SESSION_TYPE")
	return s == "wayland"
}

// Gets the subsystem host ip
// If the CLI is running under WSL the localhost url will not work so
// this function should return the real ip that we should redirect to
func GetWslHostName() string {
	out, err := exec.Command("wsl.exe", "hostname", "-I").Output()
	if err != nil {
		return "localhost"
	}
	return strings.TrimSpace(string(out))
}

func (c Commands) List() error {
	its, err := c.GetAllFeeds()
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	output := ""

	for _, item := range its {
		output += fmt.Sprintf("%s \n  - %s\n", item.Title, item.Link)
	}

	if c.config.Pager == "false" {
		fmt.Println(output)
		return nil
	}

	return outputToPager(output)
}

func (c Commands) Add(url string, name string, tags []string) error {
	err := c.config.AddFeed(config.Feed{URL: url, Name: name, Tags: tags})
	if err != nil {
		return fmt.Errorf("commands Add: %w", err)
	}

	return nil
}

func (c Commands) Refresh() error {
	_, _, err := c.fetchAllFeeds()
	if err != nil {
		return fmt.Errorf("commands Refresh: %w", err)
	}

	return nil
}

func (c Commands) ShowConfig() error {
	yaml, err := yaml.Marshal(&c.config)
	if err != nil {
		return fmt.Errorf("commands Config: %w", err)
	}
	fmt.Print(string(yaml))
	return nil
}

func (c Commands) ImportFeeds(source string) error {
	var opmlData []byte
	URL, err := url.Parse(source)
	if err == nil && URL.Host != "" && URL.Scheme != "" {
		fmt.Println("Fetch OPML from remote URL: " + URL.String())
		res, err := http.Get(URL.String())
		if err != nil {
			return fmt.Errorf("config.ImportFeeds: opml fetch error: %w", err)
		}
		opmlData, err = io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("config.ImportFeeds: error reading opml body: %w", err)
		}
	} else {
		fmt.Println("Read OMPL from file: " + source)
		opmlData, err = os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("config.ImportFeeds: error reading opml from file: %w", err)
		}
	}
	opml, err := parseOPML(opmlData)
	if err != nil {
		return fmt.Errorf("config.ImportFeeds: error parsing OPML: %w", err)
	}
	feeds := make([]config.Feed, 0)
	for _, outline := range opml.Body.Outlines {
		if outline.XMLUrl == nil {
			log.Printf("config.ImportFeeds: No url for outline %s\n", outline.Title)
		} else {
			feeds = append(feeds, config.Feed{
				Name: outline.Title,
				URL:  outline.XMLUrl.String(),
			})
		}

		feeds = slices.Concat(feeds, getChildFeeds(outline))
	}

	errors := 0
	for _, feed := range feeds {
		err := c.config.AddFeed(feed)
		if err != nil {
			errors++
			log.Printf("config.ImportFeeds: %s\n", err)
		}
	}

	fmt.Printf("added %d feeds with %d errors", len(feeds)-errors, errors)

	return nil
}

func getChildFeeds(outline Outline) []config.Feed {
	feeds := make([]config.Feed, 0)
	for _, child := range outline.Outlines {
		if child.XMLUrl == nil {
			log.Printf("getChildFeeds: No url for outline %s\n", child.Title)
		} else {
			feeds = append(feeds, config.Feed{
				Name: child.Title,
				URL:  child.XMLUrl.String(),
			})
		}

		feeds = slices.Concat(feeds, getChildFeeds(child))
	}

	return feeds
}

type FetchResultError struct {
	res rss.RSS
	err error
	url string
}

type ErrorItem struct {
	FeedURL string
	Err     error
}

func (c Commands) fetchAllFeeds() ([]store.Item, []ErrorItem, error) {
	var (
		items      []store.Item
		wg         sync.WaitGroup
		errorItems []ErrorItem
	)

	feeds := c.config.GetFeeds()

	if len(feeds) <= 0 {
		return items, errorItems, fmt.Errorf("no feeds found, add to nom/config.yml")
	}

	ch := make(chan FetchResultError)

	for _, feed := range feeds {
		wg.Add(1)

		go fetchFeed(ch, &wg, feed, c.config.HTTPOptions, c.config.Version)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	err := c.store.BeginBatch()
	if err != nil {
		return items, errorItems, fmt.Errorf("fetchAllFeeds: failed to begin batch: %w", err)
	}
	defer c.store.EndBatch()

	for result := range ch {
		if result.err != nil {
			errorItems = append(errorItems, ErrorItem{FeedURL: result.url, Err: result.err})
			continue
		}

		for _, r := range result.res.Channel.Items {
			i := store.Item{
				Author:      r.Author,
				Content:     r.Content,
				FeedURL:     result.url,
				FeedName:    r.FeedName,
				Link:        r.Link,
				GUID:        r.GUID,
				PublishedAt: r.PubDate,
				Title:       r.Title,
			}

			err := c.store.UpsertItem(&i)
			if err != nil {
				log.Printf("[commands.go] fetchAllFeeds: failed to upsert item: %v", err)
				continue
			}

			items = append(items, i)
		}
	}

	return items, errorItems, nil
}

// Monitor refreshes the feeds every RefreshInterval minutes, sending the
// TUI what it found, until ctx ends.
func (c Commands) Monitor(ctx context.Context, send func(limoni.Msg)) {
	if c.config.RefreshInterval == 0 {
		return
	}

	go func() {
		t := time.NewTicker(time.Duration(c.config.RefreshInterval) * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			err := c.Refresh()
			if err != nil {
				log.Println("Refresh failed: ", err)
				send(statusUpdate{status: "Refresh failed"})
				continue
			}
			items, err := c.GetAllFeeds()
			if err != nil {
				log.Println("Refresh failed: ", err)
				send(statusUpdate{status: "Refresh failed"})
			}
			send(listUpdate{items: convertItems(items), status: "Refreshed."})
		}
	}()
}

func (c Commands) CountUnread() int {
	count, err := c.store.CountUnread()
	if err != nil {
		log.Println(err)
	}
	return count
}

// GetArticleMarkdown is the article as the article view shows it: its
// title, author, date and link, then its content as Markdown.
func (c Commands) GetArticleMarkdown(ID int) (string, error) {
	article, err := c.store.GetItemByID(ID)
	if err != nil {
		return "", fmt.Errorf("commands.GetArticleMarkdown: %w", err)
	}

	if c.config.AutoRead && !article.Read() {
		err = c.store.ToggleRead(article.ID)
		if err != nil {
			return "", fmt.Errorf("[commands.go] GetArticleMarkdown: %w", err)
		}
	}

	return itemMarkdown(article, c.config.Theme), nil
}

func itemMarkdown(item store.Item, theme config.Theme) string {
	var mdown string

	title := item.Title
	if item.Read() {
		title = fmt.Sprintf("%s - %s", item.Title, theme.ReadIcon)
	}

	mdown += "# " + title
	mdown += "\n"
	mdown += item.Author
	if !item.PublishedAt.IsZero() {
		mdown += "\n"
		mdown += item.PublishedAt.String()
	}
	mdown += "\n\n"
	mdown += item.Link
	mdown += "\n\n"
	mdown += htmlToMd(item.Content, item.Link)

	return mdown
}

// htmlToMd converts an article's HTML to Markdown. Relative addresses in it
// are resolved against base, the article's own address, so that its links
// work and its pictures can be fetched.
func htmlToMd(html string, base string) string {
	baseURL, _ := url.Parse(base)
	converter := md.NewConverter(md.DomainFromURL(base), true, &md.Options{
		GetAbsoluteURL: func(_ *goquery.Selection, raw string, _ string) string {
			ref, err := url.Parse(raw)
			if err != nil || baseURL == nil || baseURL.Host == "" || strings.HasPrefix(raw, "#") || ref.Scheme == "data" {
				return raw
			}
			return baseURL.ResolveReference(ref).String()
		},
	})

	mdown, err := converter.ConvertString(html)
	if err != nil {
		log.Printf("[commands.go] htmlToMd: failed to convert HTML to markdown: %v", err)
		// Return the original HTML if conversion fails
		return html
	}

	return mdown
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
