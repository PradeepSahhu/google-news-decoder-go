package main

import (
	"flag"
	"fmt"
	"os"

	googlenewsdecoder "github.com/PradeepSahhu/google-news-decoder-go"
)

func main() {
	proxy := flag.String("proxy", "", "Optional proxy URL")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: google-news-decoder [-proxy http://host:port] <google-news-url> [more-urls...]")
		os.Exit(2)
	}

	decoder, err := googlenewsdecoder.NewGoogleDecoder(*proxy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create decoder: %v\n", err)
		os.Exit(1)
	}

	for _, sourceURL := range flag.Args() {
		decoded, err := decoder.DecodeGoogleNewsUrl(sourceURL, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s -> error: %v\n", sourceURL, err)
			continue
		}
		fmt.Printf("%s -> %s\n", sourceURL, decoded)
	}
}
