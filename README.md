# google-news-decoder-go

A Go library (and CLI) to decode Google News article URLs into their original source URLs.

## Install as a dependency

```bash
go get github.com/PradeepSahhu/google-news-decoder-go@latest
```

## Use in another Go project

```go
package main

import (
	"fmt"

	googlenewsdecoder "github.com/PradeepSahhu/google-news-decoder-go"
)

func main() {
	decoder, err := googlenewsdecoder.NewGoogleDecoder("")
	if err != nil {
		panic(err)
	}

	source := "https://news.google.com/articles/..."
	decoded, err := decoder.DecodeGoogleNewsUrl(source, 0)
	if err != nil {
		panic(err)
	}

	fmt.Println(decoded)
}
```

## CLI usage

Run without installing:

```bash
go run ./cmd/google-news-decoder -- "https://news.google.com/articles/..."
```

Install CLI binary:

```bash
go install github.com/PradeepSahhu/google-news-decoder-go/cmd/google-news-decoder@latest
```

Then use:

```bash
google-news-decoder "https://news.google.com/articles/..."
```

Optional proxy support:

```bash
google-news-decoder -proxy "http://127.0.0.1:8080" "https://news.google.com/articles/..."
```
