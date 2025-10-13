package commands

import (
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
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
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

func convertItems(its []store.Item) []list.Item {
	var items []list.Item

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

func (c Commands) Add(url string, name string) error {
	err := c.config.AddFeed(config.Feed{URL: url, Name: name})
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
				PublishedAt: r.PubDate,
				Title:       r.Title,
			}

			id, err := c.store.UpsertItem(i)
			if err != nil {
				log.Fatalf("[commands.go] fetchAllFeeds: %e", err)
				continue
			}

			i.ID = id

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

func (c Commands) Monitor(prog *tea.Program) {
	if c.config.RefreshInterval == 0 {
		return
	}

	go func() {
		t := time.NewTicker(time.Duration(c.config.RefreshInterval) * time.Minute)
		for range t.C {
			err := c.Refresh()
			if err != nil {
				log.Println("Refresh failed: ", err)
				prog.Send(statusUpdate{
					status: "Refresh failed",
				})
			} else {
				items, err := c.GetAllFeeds()
				if err != nil {
					log.Println("Refresh failed: ", err)
					prog.Send(statusUpdate{
						status: "Refresh failed",
					})
				}
				prog.Send(listUpdate{
					items:  convertItems(items),
					status: "Refreshed.",
				})
			}
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

func (c Commands) GetGlamourisedArticle(ID int) (string, error) {
	article, err := c.store.GetItemByID(ID)
	if err != nil {
		return "", fmt.Errorf("commands.FindGlamourisedArticle: %w", err)
	}

	if c.config.AutoRead && !article.Read() {
		err = c.store.ToggleRead(article.ID)
		if err != nil {
			return "", fmt.Errorf("[commands.go] GetGlamourisedArticle: %w", err)
		}
	}

	content, err := glamouriseItem(article, c.config.Theme)
	if err != nil {
		return "", fmt.Errorf("[commands.go] GetGlamourisedArticle: %w", err)
	}

	return content, nil
}

func getStyleConfigWithOverrides(theme config.Theme) (sc ansi.StyleConfig) {
	switch theme.Glamour {
	case "light":
		sc = glamour.LightStyleConfig
	case "dracula":
		sc = glamour.DraculaStyleConfig
	case "pink":
		sc = glamour.PinkStyleConfig
	case "ascii":
		sc = glamour.ASCIIStyleConfig
	case "notty":
		sc = glamour.NoTTYStyleConfig
	default:
		sc = glamour.DarkStyleConfig
	}

	sc.H1.BackgroundColor = &theme.TitleColor
	sc.H1.Color = &theme.TitleColorFg

	return sc
}

func glamouriseItem(item store.Item, theme config.Theme) (string, error) {
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
	mdown += htmlToMd(item.Content)

	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(getStyleConfigWithOverrides(theme)),
	)

	out, err := r.Render(mdown)
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
