# nom
> Feed me

![](./.github/demo.gif)

## Install
```sh
$ go install github.com/guyfedwards/nom@latest
```

See [releases](https://github.com/guyfedwards/nom/releases) for binaries

## Config
Add feeds with the `add` command 
```sh
$ nom add <url>
```
or add directly to the config at `~/.config/nom/config.yml`.
```yaml
feeds:
  - url: https://dropbox.tech/feed
  - url: https://snyk.io/blog/feed
```

## Usage
```sh
$ nom # open TUI
$ nom list -n 20 # optionally show more
$ nom add <feed_url> 
$ nom read <title_substring> 
$ nom list --no-cache # fetch new results
```
