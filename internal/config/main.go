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
	URL string `yaml:"url"`
}

type Config struct {
	configPath string
	Pager      string `yaml:"pager,omitempty"`
	NoCache    bool   `yaml:"no-cache,omitempty"`
	Feeds      []Feed `yaml:"feeds"`
	Preview    string `yaml:"preview,omitempty"`
}

func New(configPath string, pager string, noCache bool, preview string) (Config, error) {
	if configPath == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			return Config{}, fmt.Errorf("config.New: %w", err)
		}

		configPath = filepath.Join(userConfigDir, "nom/config.yml")
	}

	return Config{
		configPath: configPath,
		Pager:      pager,
		NoCache:    noCache,
		Feeds:      []Feed{},
		Preview:    preview,
	}, nil
}

func (c *Config) Load() error {
	err := setupConfigDir(c.configPath)
	if err != nil {
		return fmt.Errorf("config Load: %w", err)
	}

	rawData, err := os.ReadFile(c.configPath)
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}

	var fileConfig Config
	err = yaml.Unmarshal(rawData, &fileConfig)
	if err != nil {
		return fmt.Errorf("config.Read: %w", err)
	}

	c.Feeds = fileConfig.Feeds
	// only set pager if it's not defined already, config file is lower
	// precidence than flags/env that can be passed to New
	if c.Pager == "" {
		c.Pager = fileConfig.Pager
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

	c.Feeds = append(c.Feeds, feed)

	err = c.Write()
	if err != nil {
		return fmt.Errorf("config.AddFeed: %w", err)
	}

	return nil
}

func setupConfigDir(configPath string) error {
	// if configpath already exists, exit early
	if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	pieces := strings.Split(configPath, "/")
	path := strings.Join(pieces[:len(pieces)-1], "/")

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	_, err = os.Create(configPath)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	return nil
}
