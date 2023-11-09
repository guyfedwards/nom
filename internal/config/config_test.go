package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/guyfedwards/nom/internal/test"
)

const configFixturePath = "../test/data/config_fixture.yml"
const configFixtureWritePath = "../test/data/config_fixture_write.yml"
const configDir = "../test/data/nom"
const configPath = "../test/data/nom/config.yml"

func cleanup() {
	os.RemoveAll(configDir)
}

func TestNewDefault(t *testing.T) {
	c, _ := New("", "", []string{})
	ucd, _ := os.UserConfigDir()

	test.Equal(t, fmt.Sprintf("%s/nom/config.yml", ucd), c.configPath, "Wrong defaults set")
	test.Equal(t, fmt.Sprintf("%s/nom", ucd), c.ConfigDir, "Wrong default ConfigDir set")
}

func TestConfigCustomPath(t *testing.T) {
	c, _ := New("foo/bar.yml", "", []string{})

	test.Equal(t, "foo/bar.yml", c.configPath, "Config path override not set")
}

func TestConfigDir(t *testing.T) {
	c, _ := New("foo/bizzle/bar.yml", "", []string{})

	test.Equal(t, "foo/bizzle", c.ConfigDir, "ConfigDir not correctly parsed")
}

func TestNewOverride(t *testing.T) {
	c, _ := New("foobar", "", []string{})

	test.Equal(t, "foobar", c.configPath, "Override not respected")
}

func TestPreviewFeedsOverrideFeedsFromConfigFile(t *testing.T) {
	c, _ := New(configFixturePath, "", []string{})
	c.Load()
	feeds := c.GetFeeds()
	test.Equal(t, 3, len(feeds), "Incorrect feeds number")
	test.Equal(t, "cattle", feeds[0].URL, "First feed in a config must be cattle")
	test.Equal(t, "bird", feeds[1].URL, "Second feed in a config must be bird")
	test.Equal(t, "dog", feeds[2].URL, "Third feed in a config must be dog")

	c, _ = New(configFixturePath, "", []string{"pumpkin", "radish"})
	c.Load()
	feeds = c.GetFeeds()
	test.Equal(t, 2, len(feeds), "Incorrect feeds number")
	test.Equal(t, "pumpkin", feeds[0].URL, "First feed in a config must be pumpkin")
	test.Equal(t, "radish", feeds[1].URL, "Second feed in a config must be radish")
}

func TestConfigLoad(t *testing.T) {
	c, _ := New(configFixturePath, "", []string{})
	err := c.Load()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if len(c.Feeds) != 3 || c.Feeds[0].URL != "cattle" {
		t.Fatalf("Parsing failed")
	}
}

func TestConfigLoadPrecidence(t *testing.T) {
	c, _ := New(configFixturePath, "testpager", []string{})

	err := c.Load()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if c.Pager != "testpager" {
		t.Fatalf("testpager overridden")
	}
}

func TestConfigAddFeed(t *testing.T) {
	c, _ := New(configFixtureWritePath, "", []string{})

	err := c.Load()
	if err != nil {
		t.Fatalf(err.Error())
	}

	c.AddFeed(Feed{URL: "foo"})

	var actual Config
	rawData, _ := os.ReadFile(c.configPath)
	_ = yaml.Unmarshal(rawData, &actual)

	hasAdded := false
	for _, v := range actual.Feeds {
		if v.URL == "newfeed" {
			hasAdded = true
			break
		}
	}

	if !hasAdded {
		t.Fatalf("did not write feed correctly")
	}
}

func TestConfigSetupDir(t *testing.T) {
	err := setupConfigDir(configDir)
	if err != nil {
		t.Fail()
		return
	}

	_, err = os.Stat(filepath.Join(configDir, "config.yml"))
	if err != nil {
		t.Fail()
	}

	cleanup()
}
