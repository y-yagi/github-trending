# Github Trending

This is tool for seeing GitHub Trending.

[![asciicast](https://asciinema.org/a/aRVVXUVQk3unOHfmdSdVLmh1j.png)](https://asciinema.org/a/aRVVXUVQk3unOHfmdSdVLmh1j)

## Usage

The language to be showed can be specified by config. By default only `All languages` is showed.  config can be edited with the `-c` option. Example is as follows.

```toml
languages = ["all", "go", "ruby", "javascript"]
browser = "google-chrome"
```

After specifying the repository and pressing enter, you can access the repository in browser. You can specify the browser to use `browser` config.


## Installation


```
$ go get github.com/y-yagi/github-trending
```
