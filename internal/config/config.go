package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Backends struct {
	Miniflux *MinifluxBackend `yaml:"miniflux,omitempty"`
	FreshRSS *FreshRSSBackend `yaml:"freshrss,omitempty"`
}

type Opener struct {
	Regex string `yaml:"regex"`
	Cmd   string `yaml:"cmd"`
}

// need to add to Load() below if loading from config file
type Config struct {
	configPath string
	ConfigDir  string `yaml:"-"`
	Pager      string `yaml:"pager,omitempty"`
	Feeds      []Feed `yaml:"feeds"`
	// Preview feeds are distinguished from Feeds because we don't want to inadvertenly write those into the config file.
	PreviewFeeds   []Feed    `yaml:"previewfeeds,omitempty"`
	Backends       *Backends `yaml:"backends,omitempty"`
	ShowRead       bool      `yaml:"showread,omitempty"`
	AutoRead       bool      `yaml:"autoread,omitempty"`
	ShowFavourites bool
	Openers        []Opener `yaml:"openers,omitempty"`
}

func (c *Config) ToggleShowRead() {
	c.ShowRead = !c.ShowRead
}

func (c *Config) ToggleShowFavourites() {
	c.ShowFavourites = !c.ShowFavourites
}

func New(configPath string, pager string, previewFeeds []string) (Config, error) {
	var configDir string

	if configPath == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			return Config{}, fmt.Errorf("config.New: %w", err)
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

	return Config{
		configPath:   configPath,
		ConfigDir:    configDir,
		Pager:        pager,
		Feeds:        []Feed{},
		PreviewFeeds: f,
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

	rawData, err := os.ReadFile(c.configPath)
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
	c.Openers = fileConfig.Openers
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

	err = os.WriteFile(c.configPath, []byte(str), 0655)
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
			return errors.New("config.AddFeed: feed already exists")
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
