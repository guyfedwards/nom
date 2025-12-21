# nom

> Feed me

`nom` is a terminal based RRS feed reader using [Glow](https://github.com/charmbracelet/glow) styled markdown to improve the reading experience and a simple TUI using [Bubbletea](https://github.com/charmbracelet/bubbletea).

- Local sync and offline reading
- Backend connections (miniflux, freshrss supported)
- Vim style keybindings for navigation
- Plenty more features such as mark read/unread, filtering and feed naming

![](./.github/demo.gif)

## Install

See [releases](https://github.com/guyfedwards/nom/releases) for binaries. E.g.

```sh
curl -L https://github.com/guyfedwards/nom/releases/download/v2.16.2/nom_2.16.2_darwin_amd64.tar.gz | tar -xzvf -
```

To install the `nom` binary into `/usr/local/bin` (or into the location of your choice) in a single step:

```sh
curl -L https://github.com/guyfedwards/nom/releases/download/v2.16.2/nom_2.16.2_darwin_amd64.tar.gz |
  sudo tar -C /usr/local/bin -xvzf - nom
```

## Usage

```sh
nom # start TUI
nom add <feed_url> <optional feed_name>
nom -h # see all available command and options
```

## Configuration

Configuration lives by default in `$XDG_CONFIG_HOME/nom/config.yml` or `$HOME/Library/Application Support/nom/config.yml` on darwin. You can customise the location of the configuration file with the `--config-path` (`-c`) flag:

```sh
nom -c my-custom-config.yml
```

### Feeds

Feeds are listed in the `feeds` section of the configuration file. They have a URL, an option name, and an optional list of tags:

```yaml
feeds:
- url: https://dropbox.tech/feed
  name: DropBox
- url: https://snyk.io/blog/feed
  name: Snyk
  tags:
    - ai
    - tech
```

You can also add feeds with the `add` command:

```sh
nom add [-n <feed name>] [-t tag [...]] <url>
```

Feeds are editable within `nom` by pressing `E` to open the configuration in your editor. You can configure which editor Nom will use by setting (in order of preference) your `$NOMEDITOR`, `$VISUAL`, or `$EDITOR` environment variable. After editing feeds, you will need to then refresh with `r`.

Alternatively you can import feeds from an OPML file:

```sh
nom import <path/to/opml|url/to/opm>
```

#### YouTube feeds

To add YouTube feeds you can go to a channel and run the following in the browser console to get the rss feed link:

```js
console.log(
  `https://www.youtube.com/feeds/videos.xml?channel_id=${
    document.querySelector("link[rel='canonical']").href.split("/channel/")
      .reverse()[0]
  }`,
);
```

### Show read (default: false)

Show read items by default. (can be toggled with M)

```yaml
showread: true
```

### Auto read (default: false)

Automatically mark items as read on selection or navigation through items.

```yaml
autoread: true
```

### Ordering

Set the default sort ordering of the list

```yaml
ordering: asc
```

### Filtering

Default to include the feedname prefix in filtering query. Removes need to use `f:xxx` for simple queries. This will mean that multi-feed filters won't work, e.g. `f:xxx f:yyy`

```yaml
filtering:
  defaultIncludeFeedName: true
```

### Refresh interval

Background refresh interval in minutes. Setting this to anything but 0 will make `nom` refresh automatically.

```yaml
refreshinterval: 5
```

### Theme

Theme allows some basic color overrides in the feed view and then setting a custom markdown render theme for the overall markdown view. `theme.glamour` can be one of "dark", "dracula", "light", "pink", "ascii" or "notty". See [here](https://github.com/charmbracelet/glamour/tree/master/styles/gallery) for previews and more info.
Colors can be hex or ASCII codes, they will be coerced depending on your terminal color settings.

```yaml
theme:
  glamour: dark
  titleColor: "62"
  titleColorFg: "231"
  selectedItemColor: "170"
  filterColor: "#555555"
```

### Backends

As well as adding feeds directly, you can pull in feeds from another source. You can add multiple backends and the feeds will all be added.

```yaml
backends:
  miniflux:
    host: http://myminiflux.foo
    api_key: jafksdljfladjfk
  freshrss:
    host: http://myfreshrss.bar
    user: admin
    password: muchstrong
    prefixCats: true # prefix feed name for freshrss entries
```

#### FreshRSS

To use freshrss you need to enable API access and set the API password explicitly, separate to your user password.

1. To enable the API go to Settings > Authentication > Allow API access.
1. You can set the API password in Settings > Profile > API password.

### Openers

By default links are opened in the browser, you can specify commands to open certain links based on a regex string.\
`regex` can be any valid golang regex string, it will be matched against the feed item link.\
`cmd` is run as a child command. The `%s` denotes the position of the link in the command.\
`takeover` dictates if the command should takeover the tty from nom. E.g. for opening links in lynx or other TUI.

```yaml
openers:
- regex: "youtube"
  cmd: "mpv %s"
- regex: ".*"
  cmd: "lynx %s"
  takeover: true
```

### Proxy support

If you need to use a proxy server for internet access, you can configure `nom`
by setting the environment variables `HTTP_PROXY` and `HTTPS_PROXY` to point to
your proxy server:

```sh
export HTTP_PROXY=https://proxy.example.com
export HTTPS_PROXY=https://proxy.example.com
nom
```

From the [ProxyFromEnvironment documentation](https://pkg.go.dev/net/http#ProxyFromEnvironment):

> [Use a proxy] as indicated by the environment variables `HTTP_PROXY`,
> `HTTPS_PROXY` and `NO_PROXY` (or the lowercase versions thereof). Requests use
> the proxy from the environment variable matching their scheme, unless
> excluded by `NO_PROXY`.
>
> The environment values may be either a complete URL or a "host[:port]", in
> which case the "http" scheme is assumed.

## Store

Nom uses sqlite as a store for feeds and metadata. It is stored adjacent to the configuration file in `$XDG_CONFIG_HOME/nom/nom.db`. This can be backed up like any file and will store articles, read state etc. It can also be deleted to start from scratch, re-downloading all articles and no state.

The name of the sqlite file can be overridden in the configuration file, allowing you to have multiple configurations each with their own data store.

```yaml
database: news.db
```

## Filtering

Within the `nom` view, you can filter by title pressing the `/` character. Nom supports simple keyword searches as well as searches using the `feed:` and `tag:` qualifiers.

### Simple keyword searches

- `example` will match any titles that contain the word `example`.
-  If you have `defaultIncludeFeedName: true` in your nom configuration, this will also match any items whose feed name contains `example`.

### Feed name searches

You can limit results to feeds with a certain pattern in their name using the `feed:` qualifier:

- `feed:example` or `f:example` will match any titles from a feed that contains `example` in the feed name.
- If your feed name contains spaces, you can quote the name using single or double quotes:

  - `feed:"example feed"`
  - `feed:'example feed'`

  Or you can backslash-escape the spaces:

  - `feed:example\ feed`

### Tag searches

You can limit results to feed that have certain tags using the `tag:` qualifier:

- `tag:example` or `t:example` will match titles from feed that has the tag `example`.

### Include feedname in filtering

If you want to include the feed name in the default filtering query, use `config.filtering.defaultIncludeFeedName: true`. This simplifies the above `f:xxx` queries but means that you can't filter by multiple feeds at once, e.g. `f:xxx f:yyy`.

### Filter styles cannot be combined

In the current implementation, you *cannot* combine different types of filters. That is, you can do this:

- `feed:foo feed:bar`
- `tag:foo tag:bar`

But you cannot combine `feed:` and `tag:` queries:

- `feed:foo tag:news` will return all results that match `feed:foo` and will ignore the `tag:` qualifier.

And you cannot combine simple keyword searches with any qualifiers:

- `tag:news boston` will return all results that match `tag:news` and will ignore the additional keyword.

## Building and Running via Docker

Build nom image

```sh
docker build -t nom .
```

This embeds the local `docker-config.yml`` file into the container and will be used by default.

Running Nom via docker

```sh
docker run --rm -it nom
```

Use the `-v` command line argument to mount a local configuration onto `/app/docker-config.yml` as desired:

```sh
docker run --rm -it -v $PWD/my-nom-config.yml:/app/docker-config.yml nom
```

## Dev setup

You can use `backends-compose.yml` to spin up a local instance of [MiniFlux] and [FreshRSS] if needed for development.

[miniflux]: https://miniflux.app/
[freshrss]: https://www.freshrss.org/

```sh
docker-compose -f backends-compose.yml up -d
```

### Debug logging

You can enable logging to a file using the `DEBUGNOM` env var. Passing any path will cause `log` calls to write there e.g. `DEBUGNOM=debug.log`
