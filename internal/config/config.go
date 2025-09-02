package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/guyfedwards/nom/v2/internal/constants"
)

var (
	ErrFeedAlreadyExists = errors.New("config.AddFeed: feed already exists")
)

type Feed struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name,omitempty"`
}

type MinifluxBackend struct {
	Host   string `yaml:"host"`
	APIKey string `yaml:"api_key"`
}

type FreshRSSBackend struct {
	Host       string `yaml:"host"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	PrefixCats bool   `yaml:"prefixCats"`
}

type Backends struct {
	Miniflux *MinifluxBackend `yaml:"miniflux,omitempty"`
	FreshRSS *FreshRSSBackend `yaml:"freshrss,omitempty"`
}

type Opener struct {
	Regex    string `yaml:"regex"`
	Cmd      string `yaml:"cmd"`
	Takeover bool   `yaml:"takeover"`
}

type Theme struct {
	Glamour           string `yaml:"glamour,omitempty"`
	TitleColor        string `yaml:"titleColor,omitempty"`
	TitleColorFg      string `yaml:"titleColorFg,omitempty"`
	FilterColor       string `yaml:"filterColor,omitempty"`
	SelectedItemColor string `yaml:"selectedItemColor,omitempty"`
	ReadIcon          string `yaml:"readIcon,omitempty"`
}

type FilterConfig struct {
	DefaultIncludeFeedName bool `yaml:"defaultIncludeFeedName"`
}

// need to add to Load() below if loading from config file
type Config struct {
	ConfigPath     string
	ShowFavourites bool `yaml:"showfavourites,omitempty"`
	Version        string
	ConfigDir      string       `yaml:"-"`
	Pager          string       `yaml:"pager,omitempty"`
	Feeds          []Feed       `yaml:"feeds"`
	Database       string       `yaml:"database"`
	Ordering       string       `yaml:"ordering"`
	Filtering      FilterConfig `yaml:"filtering"`
	// Preview feeds are distinguished from Feeds because we don't want to inadvertenly write those into the config file.
	PreviewFeeds []Feed       `yaml:"previewfeeds,omitempty"`
	Backends     *Backends    `yaml:"backends,omitempty"`
	ShowRead     bool         `yaml:"showread,omitempty"`
	AutoRead     bool         `yaml:"autoread,omitempty"`
	Openers      []Opener     `yaml:"openers,omitempty"`
	Theme        Theme        `yaml:"theme,omitempty"`
	HTTPOptions  *HTTPOptions `yaml:"http,omitempty"`
}

func (c *Config) ToggleShowRead() {
	c.ShowRead = !c.ShowRead
}

func (c *Config) ToggleShowFavourites() {
	c.ShowFavourites = !c.ShowFavourites
}

func New(configPath string, pager string, previewFeeds []string, version string) (*Config, error) {
	var configDir string

	if configPath == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("config.New: %w", err)
		}

		configDir = filepath.Join(userConfigDir, "nom")
		configPath = filepath.Join(configDir, "/config.yml")
	} else {
		// strip off end of path as config filename
		sep := string(os.PathSeparator)
		parts := strings.Split(configPath, sep)
		configDir = strings.Join(parts[0:len(parts)-1], sep)
	}

	var f []Feed
	for _, feedURL := range previewFeeds {
		f = append(f, Feed{URL: feedURL})
	}

	return &Config{
		ConfigPath:   configPath,
		ConfigDir:    configDir,
		Pager:        pager,
		Database:     "nom.db",
		Feeds:        []Feed{},
		PreviewFeeds: f,
		Theme: Theme{
			Glamour:           "dark",
			SelectedItemColor: "170",
			TitleColor:        "62",
			TitleColorFg:      "231",
			FilterColor:       "62",
			ReadIcon:          "\u2713",
		},
		Ordering: constants.DefaultOrdering,
		Filtering: FilterConfig{
			DefaultIncludeFeedName: false,
		},
		HTTPOptions: &HTTPOptions{
			MinTLSVersion: tls.VersionName(tls.VersionTLS12),
		},
	}, nil
}

func (c *Config) IsPreviewMode() bool {
	return len(c.PreviewFeeds) > 0
}

func (c *Config) Load() error {
	err := setupConfigDir(c.ConfigDir)
	if err != nil {
		return fmt.Errorf("config Load: %w", err)
	}

	rawData, err := os.ReadFile(c.ConfigPath)
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}

	// manually set config values from fileconfig, messy solve for config priority
	var fileConfig Config
	err = yaml.Unmarshal(rawData, &fileConfig)
	if err != nil {
		return fmt.Errorf("config.Read: %w", err)
	}

	c.ShowRead = fileConfig.ShowRead
	c.AutoRead = fileConfig.AutoRead
	c.Feeds = fileConfig.Feeds
	c.Database = fileConfig.Database
	c.Openers = fileConfig.Openers
	c.ShowFavourites = fileConfig.ShowFavourites
	c.Filtering = fileConfig.Filtering

	if fileConfig.HTTPOptions != nil {
		if _, err := TLSVersion(fileConfig.HTTPOptions.MinTLSVersion); err != nil {
			return err
		}
		c.HTTPOptions = fileConfig.HTTPOptions
	}

	if len(fileConfig.Ordering) > 0 {
		c.Ordering = fileConfig.Ordering
	}

	if len(fileConfig.Theme.ReadIcon) > 0 {
		c.Theme.ReadIcon = fileConfig.Theme.ReadIcon
	}

	if fileConfig.Theme.Glamour != "" {
		c.Theme.Glamour = fileConfig.Theme.Glamour
	}

	if fileConfig.Theme.SelectedItemColor != "" {
		c.Theme.SelectedItemColor = fileConfig.Theme.SelectedItemColor
	}

	if fileConfig.Theme.TitleColor != "" {
		c.Theme.TitleColor = fileConfig.Theme.TitleColor
	}

	if fileConfig.Theme.TitleColorFg != "" {
		c.Theme.TitleColorFg = fileConfig.Theme.TitleColorFg
	}

	if fileConfig.Theme.FilterColor != "" {
		c.Theme.FilterColor = fileConfig.Theme.FilterColor
	}

	// only set pager if it's not defined already, config file is lower
	// precidence than flags/env that can be passed to New
	if c.Pager == "" {
		c.Pager = fileConfig.Pager
	}

	if fileConfig.Backends != nil {
		if fileConfig.Backends.Miniflux != nil {
			mffeeds, err := getMinifluxFeeds(fileConfig.Backends.Miniflux)
			if err != nil {
				return err
			}

			c.Feeds = append(c.Feeds, mffeeds...)
		}

		if fileConfig.Backends.FreshRSS != nil {
			freshfeeds, err := getFreshRSSFeeds(fileConfig.Backends.FreshRSS)
			if err != nil {
				return err
			}

			c.Feeds = append(c.Feeds, freshfeeds...)
		}
	}

	return nil
}

// Write writes to a config file
func (c *Config) Write() error {
	str, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config.Write: %w", err)
	}

	err = os.WriteFile(c.ConfigPath, []byte(str), 0655)
	if err != nil {
		return fmt.Errorf("config.Write: %w", err)
	}

	return nil
}

func (c *Config) AddFeed(feed Feed) error {
	err := c.Load()
	if err != nil {
		return fmt.Errorf("config.AddFeed: %w", err)
	}

	for _, f := range c.Feeds {
		if f.URL == feed.URL {
			return ErrFeedAlreadyExists
		}
	}

	c.Feeds = append(c.Feeds, feed)

	err = c.Write()
	if err != nil {
		return fmt.Errorf("config.AddFeed: %w", err)
	}

	return nil
}

func (c *Config) GetFeeds() []Feed {
	if c.IsPreviewMode() {
		return c.PreviewFeeds
	}

	return c.Feeds
}

func setupConfigDir(configDir string) error {
	configFile := filepath.Join(configDir, "/config.yml")

	_, err := os.Stat(configFile)

	// if configFile exists, do nothing
	if !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// if not, create directory. noop if directory exists
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	// then create the file
	_, err = os.Create(configFile)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	return err
}
