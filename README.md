# go-fltk-sane

Allows you to leverage the `sane` framework to scan documents with a scanner.

Runs an FLTK-based UI that is light on memory.

## Screenshots

![Landscape screenshot](./docs/screenshot-landscape.png)

![Portrait screenshot](./docs/screenshot-portrait.png)

## Usage

Requires `CGO_ENABLED=1` because FLTK leverages C bindings.

```bash
sudo pacman -S go sane

go get -v
go build -v

# plug in your sane-compatible scanning device
./go-fltk-sane
```

## About

This project was hacked together relatively quickly and was my first experiment with FLTK (fast light toolkit).

Originally, I leveraged another Go library that interfaces with `sane`, but unfortunately there were memory management issues (memory would not be garbage collected properly on each document scan action).

Instead, I had to rewrite much of the application to instead parse the CLI output of `scanimage` - this mean that you need to install the `sane` package on your system and ensure that `scanimage` is available.

## Stability

This program is not foolproof. It probably will not work well unless your scanner works similar to mine.

It isn't going to detect every possible setting for your scanner, only a few defaults.

I'm not trying to boil the ocean with this; I just wanted a barebones FLTK scanner application that worked for my use case.
