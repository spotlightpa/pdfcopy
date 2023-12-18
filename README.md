# pdfcopy [![GoDoc](https://godoc.org/github.com/spotlightpa/pdfcopy?status.svg)](https://godoc.org/github.com/spotlightpa/pdfcopy) [![Go Report Card](https://goreportcard.com/badge/github.com/spotlightpa/pdfcopy)](https://goreportcard.com/report/github.com/spotlightpa/pdfcopy)

Quickie script to take screenshots of URLs for copyright submissions.

## Installation

Requirements:
- [shot-scraper](https://github.com/simonw/shot-scraper)
- pdftk
- [Go](http://golang.org).

If you just want to install the binary to your current directory and don't care about the source code, run

```bash
GOBIN="$(pwd)" go install github.com/spotlightpa/pdfcopy@latest
```
