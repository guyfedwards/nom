package config

import (
	"fmt"
	"os"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/guyfedwards/nom/internal/test"
)

const configFixturePath = "../test/data/config_fixture.yml"
const configFixtureWritePath = "../test/data/config_fixture_write.yml"

func TestNewDefault(t *testing.T) {
	c, _ := New("", "", false)
	ucd, _ := os.UserConfigDir()

	test.Equal(t, fmt.Sprintf("%s/nom/config.yml", ucd), c.configPath, "Wrong defaults set")
}

func TestNewOverride(t *testing.T) {
	c, _ := New("foobar", "", false)

	test.Equal(t, "foobar", c.configPath, "Override not respected")
}

func TestConfigLoad(t *testing.T) {
	c, _ := New(configFixturePath, "")
	err := c.Load()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if len(c.Feeds) != 3 || c.Feeds[0].URL != "cattle" {
		t.Fatalf("Parsing failed")
	}
}

func TestConfigLoadPrecidence(t *testing.T) {
	c, _ := New(configFixturePath, "testpager")
	err := c.Load()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if c.Pager != "testpager" {
		t.Fatalf("testpager overridden")
	}
}

func TestConfigAddFeed(t *testing.T) {
	c, _ := New(configFixtureWritePath, "")
	c.Load()
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
