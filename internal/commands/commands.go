package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/rss"
	"github.com/guyfedwards/nom/v2/internal/store"
)

type Commands struct {
	config config.Config
	store  store.Store
}

func New(config config.Config, store store.Store) Commands {
	return Commands{config, store}
}

func convertItems(its []store.Item) []list.Item {
	var items []list.Item

	for _, item := range its {
		items = append(items, ItemToTUIItem(item))
	}

	return items
}

func (c Commands) OpenLink(url string) error {
	for _, o := range c.config.Openers {
		match, err := regexp.MatchString(o.Regex, url)
		if err != nil {
			return fmt.Errorf("OpenLink: regex: %w", err)
		}

		if match {
			c := fmt.Sprintf(o.Cmd, url)
			parts := strings.Fields(c)

			cmd := exec.Command(parts[0], parts[1:]...)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("OpenLink: exec: %w", err)
			}
		}
	}

	return c.OpenInBrowser(url)
}

func (c Commands) OpenInBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		if IsWSL() {
			cmd = "cmd.exe"
			args = []string{"/c", "start"}
		} else {
			cmd = "xdg-open"
		}
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
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

func (c Commands) TUI() error {
	debug := os.Getenv("DEBUGNOM")
	if debug != "" {
		f, err := tea.LogToFile(debug, "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	err := c.CleanFeeds()
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	its, err := c.GetAllFeeds()
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	var errorItems []ErrorItem
	// if no feeds in store or we have preview feeds, fetchAllFeeds
	if len(its) == 0 || len(c.config.PreviewFeeds) > 0 {
		_, errorItems, err = c.fetchAllFeeds()
		if err != nil {
			return fmt.Errorf("[commands.go] TUI: %w", err)
		}

		// refetch for consistent data across calls
		its, err = c.GetAllFeeds()
		if err != nil {
			return fmt.Errorf("[commands.go] TUI: %w", err)
		}
	}

	items := convertItems(its)

	es := []string{}
	for _, e := range errorItems {
		es = append(es, fmt.Sprintf("Error fetching %s: %s", e.FeedURL, e.Err))
	}

	if err := Render(items, c, es); err != nil {
		return fmt.Errorf("commands.TUI: %w", err)
	}

	return nil
}

func (c Commands) List(numResults int) error {
	err := c.CleanFeeds()
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}
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

		go fetchFeed(ch, &wg, feed)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

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
				PublishedAt: r.PubDate,
				Title:       r.Title,
			}

			// only store if non-preview feed
			if !includes(c.config.PreviewFeeds, config.Feed{URL: result.url}) {
				err := c.store.UpsertItem(i)
				if err != nil {
					log.Fatalf("[commands.go] fetchAllFeeds: %e", err)
					continue
				}
			}

			items = append(items, i)
		}
	}

	return items, errorItems, nil
}

func includes[T comparable](arr []T, item T) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}

func (c Commands) GetAllFeeds() ([]store.Item, error) {
	is, err := c.store.GetAllItems()
	if err != nil {
		return []store.Item{}, fmt.Errorf("commands.go: GetAllFeeds %w", err)
	}

	// filter out read and add feedname
	var items []store.Item
	for i := range is {
		if c.config.ShowFavourites && !is[i].Favourite {
			continue
		}

		if !c.config.ShowRead && is[i].Read() {
			continue
		}

		for _, f := range c.config.Feeds {
			if f.URL == is[i].FeedURL {
				is[i].FeedName = f.Name
			}
		}

		items = append(items, is[i])
	}

	return items, nil
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

func (c Commands) GetArticleByID(ID int) (store.Item, error) {
	items, err := c.GetAllFeeds()
	if err != nil {
		return store.Item{}, fmt.Errorf("commands.FindArticle: %w", err)
	}

	var item store.Item
	for _, it := range items {
		if it.ID == ID {
			item = it
			break
		}
	}

	return item, nil
}

func (c Commands) FindArticle(substr string) (item store.Item, err error) {
	items, err := c.GetAllFeeds()
	if err != nil {
		return store.Item{}, fmt.Errorf("commands.FindArticle: %w", err)
	}

	regex, err := regexp.Compile(strings.ToLower(substr))
	if err != nil {
		return store.Item{}, fmt.Errorf("commands.FindArticle: regexp: %w", err)
	}

	for _, it := range items {
		// very basic string matching on title to read an article
		if regex.MatchString(strings.ToLower(it.Title)) {
			item = it
			break
		}
	}

	return item, nil
}

func (c Commands) GetGlamourisedArticle(ID int) (string, error) {
	article, err := c.GetArticleByID(ID)
	if err != nil {
		return "", fmt.Errorf("commands.FindGlamourisedArticle: %w", err)
	}

	if c.config.AutoRead {
		err = c.store.ToggleRead(article.ID)
		if err != nil {
			return "", fmt.Errorf("[commands.go] GetGlamourisedArticle: %w", err)
		}
	}

	content, err := glamouriseItem(article)
	if err != nil {
		return "", fmt.Errorf("[commands.go] GetGlamourisedArticle: %w", err)
	}

	return content, nil
}

func glamouriseItem(item store.Item) (string, error) {
	var mdown string

	mdown += "# " + item.Title
	mdown += "\n"
	mdown += item.Author
	if !item.PublishedAt.IsZero() {
		mdown += "\n"
		mdown += item.PublishedAt.String()
	}
	mdown += "\n\n"
	mdown += item.Link
	mdown += "\n\n"
	mdown += htmlToMd(item.Content)

	out, err := glamour.Render(mdown, "dark")
	if err != nil {
		return "", fmt.Errorf("GlamouriseItem: %w", err)
	}

	return out, nil
}

func htmlToMd(html string) string {
	converter := md.NewConverter("", true, nil)

	mdown, err := converter.ConvertString(html)
	if err != nil {
		log.Fatal(err)
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
