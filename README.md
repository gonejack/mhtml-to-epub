# mhtml-to-epub

Command line tool for converting mhtml to epub.

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gonejack/mhtml-to-epub)
![Build](https://github.com/gonejack/mhtml-to-epub/actions/workflows/go.yml/badge.svg)
[![GitHub license](https://img.shields.io/github/license/gonejack/mhtml-to-epub.svg?color=blue)](LICENSE)

### Install
```shell
> go get github.com/gonejack/mhtml-to-epub
```

### Usage
```shell
> mhtml-to-epub *.eml
```
```
Usage:
  mhtml-to-epub [-o output] [--title title] [--cover cover] *.mht [flags]

Flags:
      --cover string    epub cover image
      --title string    epub title (default "MHTML")
      --author string   epub author (default "MHTML to Epub")
  -o, --output string   output filename (default "output.epub")
  -v, --verbose         verbose
  -h, --help            help for mht-to-epub
```
