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
$ curl -L https://github.com/guyfedwards/nom/releases/download/v2.1.4/nom_2.1.4_darwin_amd64.tar.gz | tar -xzvf -
```

## Usage
```sh
$ nom # start TUI
$ nom list -n 20 # list feed items in $PAGER, optionally show more
$ nom add <feed_url> 
$ nom --feed <feed_url> # preview feed without adding to config
```

## Config
Config lives by default in `$XDG_CONFIG_HOME/nom/config.yml` or `$HOME/Library/Application Support/nom/config.yml` on darwin.  
You can customise the location of the config file with the `--config-path` flag.

### Feeds
Feeds are added to the config file and have a url and name.
```yaml
feeds:
  - url: https://dropbox.tech/feed
    # name will be prefixed to all entries in the list
    name: dropbox 
  - url: https://snyk.io/blog/feed
```
You can also add feeds with the `add` command:
```sh
$ nom add <url>
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
```

### Openers
By default links are opened in the browser, you can specify commands to open certain links based on a regex string.   
`regex` can be any valid golang regex string, it will be matched against the feed item link.  
`cmd` is run as a child command. The `%s` denotes the position of the link in the command.  
```yaml
openers:
  - regex: "youtube"
    cmd: "mpv %s"
```

## Store
`nom` uses sqlite as a store for feeds and metadata. It is stored next to the config in `$XDG_CONFIG_HOME/nom/nom.db`. This can be backed up like any file and will store articles, read state etc. It can also be deleted to start from scratch redownloading all articles and no state.

## Building and Running via Docker
Build nom image
```sh
docker build -t nom .
```
This embeds the local docker-config.yml file into the container and will be used by default.

Running the nom via docker
```sh
docker run --rm -it nom
```
Use the `-v` command line argument to mount a local config onto `/app/docker-config.yml` as desired.


## Dev setup
You can use the `backends-compose.yml` to spin up a local instance of miniflux and freshrss if needed for development.

```sh
$ docker-compose -f backends-compose.yml up
```
