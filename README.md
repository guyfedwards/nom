# nom
> Feed me

![](./.github/demo.gif)

## Install
```sh
$ go install github.com/guyfedwards/nom/cmd/nom@latest
```

See [releases](https://github.com/guyfedwards/nom/releases) for binaries

## Config
Add feeds with the `add` command 
```sh
$ nom add <url>
```
or add directly to the config at `~/.config/nom/config.yml` on unix systems and `$HOME/Library/Application Support/nom/config.yml` on darwin.
```yaml
feeds:
  - url: https://dropbox.tech/feed
  - url: https://snyk.io/blog/feed
```
You can customise the location of the config file with the `--config-path` flag.

## Usage
```sh
$ nom # open TUI
$ nom list -n 20 # optionally show more
$ nom add <feed_url> 
$ nom read <title_substring> 
$ nom list --no-cache # fetch new results
$ nom --feed <feed_url> # preview feed without adding to config
```
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