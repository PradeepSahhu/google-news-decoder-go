package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type GoogleDecoder struct {
	Client *http.Client
}

// NewGoogleDecoder initializes a new GoogleDecoder.
// proxyURL can be empty. If provided, it should be in the format:
// http://user:pass@host:port or socks5://user:pass@host:port
func NewGoogleDecoder(proxyURL string) (*GoogleDecoder, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if proxyURL != "" {
		pURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %v", err)
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(pURL),
		}
	}

	return &GoogleDecoder{Client: client}, nil
}

type DecodingParams struct {
	Signature string
	Timestamp string
	Base64Str string
}

// GetBase64Str extracts the base64 string from a Google News URL.
func (d *GoogleDecoder) GetBase64Str(sourceURL string) (string, error) {
	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	pathSegments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if parsedURL.Hostname() == "news.google.com" && len(pathSegments) > 1 {
		secondToLast := pathSegments[len(pathSegments)-2]
		if secondToLast == "articles" || secondToLast == "read" {
			return pathSegments[len(pathSegments)-1], nil
		}
	}

	return "", errors.New("invalid Google News URL format")
}

func (d *GoogleDecoder) getDecodingParamsFromURL(reqURL string) (*DecodingParams, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")

	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	sel := doc.Find("c-wiz > div[jscontroller]").First()
	if sel.Length() == 0 {
		return nil, errors.New("failed to fetch data attributes")
	}

	sig, ok := sel.Attr("data-n-a-sg")
	if !ok {
		return nil, errors.New("missing data-n-a-sg attribute")
	}

	ts, ok := sel.Attr("data-n-a-ts")
	if !ok {
		return nil, errors.New("missing data-n-a-ts attribute")
	}

	return &DecodingParams{
		Signature: sig,
		Timestamp: ts,
	}, nil
}

// GetDecodingParams fetches signature and timestamp required for decoding from Google News.
func (d *GoogleDecoder) GetDecodingParams(base64Str string) (*DecodingParams, error) {
	// Try the first URL format
	url1 := fmt.Sprintf("https://news.google.com/articles/%s", base64Str)
	params, err := d.getDecodingParamsFromURL(url1)
	if err == nil {
		params.Base64Str = base64Str
		return params, nil
	}

	// Try the fallback URL format
	url2 := fmt.Sprintf("https://news.google.com/rss/articles/%s", base64Str)
	params, err2 := d.getDecodingParamsFromURL(url2)
	if err2 == nil {
		params.Base64Str = base64Str
		return params, nil
	}

	return nil, fmt.Errorf("failed to get decoding params: %v (fallback error: %v)", err, err2)
}

// DecodeUrl decodes the Google News URL using the signature and timestamp.
func (d *GoogleDecoder) DecodeUrl(signature, timestamp, base64Str string) (string, error) {
	reqURL := "https://news.google.com/_/DotsSplashUi/data/batchexecute"

	innerPayload := fmt.Sprintf(`["garturlreq",[["X","X",["X","X"],null,null,1,1,"US:en",null,1,null,null,null,null,null,0,1],"X","X",1,[1,1,1],1,1,null,0,0,null,0],"%s",%s,"%s"]`, base64Str, timestamp, signature)

	payload := []interface{}{
		"Fbv4je",
		innerPayload,
	}

	reqData := [][]interface{}{{payload}}
	reqJSON, err := json.Marshal(reqData)
	if err != nil {
		return "", err
	}

	bodyStr := "f.req=" + url.QueryEscape(string(reqJSON))

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(bodyStr))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")

	resp, err := d.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bodyText := string(bodyBytes)
	parts := strings.Split(bodyText, "\n\n")
	if len(parts) < 2 {
		return "", errors.New("invalid response format")
	}

	var parsedData []interface{}
	if err := json.Unmarshal([]byte(parts[1]), &parsedData); err != nil {
		return "", fmt.Errorf("error parsing response JSON: %v", err)
	}

	if len(parsedData) == 0 {
		return "", errors.New("empty response JSON array")
	}

	firstItem, ok := parsedData[0].([]interface{})
	if !ok || len(firstItem) < 3 {
		return "", errors.New("invalid response structure (expected array with at least 3 elements)")
	}

	innerJSONStr, ok := firstItem[2].(string)
	if !ok {
		return "", errors.New("invalid response structure (expected string at index 2)")
	}

	var innerData []interface{}
	if err := json.Unmarshal([]byte(innerJSONStr), &innerData); err != nil {
		return "", fmt.Errorf("error parsing inner JSON: %v", err)
	}

	if len(innerData) < 2 {
		return "", errors.New("inner JSON array too short")
	}

	decodedURL, ok := innerData[1].(string)
	if !ok {
		return "", errors.New("decoded URL is not a string")
	}

	return decodedURL, nil
}

// DecodeGoogleNewsUrl decodes a Google News article URL into its original source URL.
func (d *GoogleDecoder) DecodeGoogleNewsUrl(sourceURL string, interval time.Duration) (string, error) {
	base64Str, err := d.GetBase64Str(sourceURL)
	if err != nil {
		return "", err
	}

	params, err := d.GetDecodingParams(base64Str)
	if err != nil {
		return "", err
	}

	decodedURL, err := d.DecodeUrl(params.Signature, params.Timestamp, params.Base64Str)
	if err != nil {
		return "", err
	}

	if interval > 0 {
		time.Sleep(interval)
	}

	return decodedURL, nil
}

func main() {
	decoder, err := NewGoogleDecoder("")
	if err != nil {
		panic(err)
	}

	sourceURL1 := "https://news.google.com/rss/articles/CBMi3gFBVV95cUxPajlzYVZfQzAwZHN0ekdHN2wtMTUzeTQ1ZVJqVDl2T2E0d1JtM0FSdUNac2VNMldVNmwzc2tVOUZtdWNCbkNpaUUwSll4cU9XTnlnNk1xeDlfaHRqMEJoUFVVeFBScVFmUjFMaEp1cTZvSmlxYWo2dWxGQ0gwcElGVlp6YjA2WTRCb3F1cVg2SDFFb25LWHlPT0NVZXFFSWFuWXZRNnVPcWNQUnRGYzhqUmJDYkJ5TTYwTm9pcVctWngteWRid0Vsa2JtS19lY2pUb1lPOTlZN1lLWWV0VXfSAeMBQVVfeXFMT1N6eGpRYS13Y0tnc1RQUlBaTG9PMkdQSnB2OFN2dTVOa2tyMVZQT0MyQ3EzbjNHNndySG1Sb3RJTFFyWG1SLUZTUF9FS3NoYktjWWV0djJhMlBncTBvTmowVEhHenpITG4zZzlqeDU4LUo2ci1NSldXdGpzMFNucV9pRkRxNTYxUzRzUExLR1V4aEVyRnNGajhnd2dZTlNNZndscmhUUDlDbEp1YlRlRF9PQU80LWFuTEV0NHA0eEEwVk1fTXVQaTB3S0xKXzFuckFoaTFVVV9VRkFWYjNuYXhwUFU?oc=5"
	sourceURL2 := "https://news.google.com/rss/articles/CBMiqwFBVV95cUxNMTRqdUZpNl9hQldXbGo2YVVLOGFQdkFLYldlMUxUVlNEaElsYjRRODVUMkF3R1RYdWxvT1NoVzdUYS0xSHg3eVdpTjdVODQ5cVJJLWt4dk9vZFBScVp2ZmpzQXZZRy1ncDM5c2tRbXBVVHVrQnpmMGVrQXNkQVItV3h4dVQ1V1BTbjhnM3k2ZUdPdnhVOFk1NmllNTZkdGJTbW9NX0k5U3E2Tkk?oc=5"

	urls := []string{sourceURL1, sourceURL2}
	var wg sync.WaitGroup

	for _, u := range urls {
		wg.Add(1)
		go func(urlStr string) {
			defer wg.Done()
			decoded, err := decoder.DecodeGoogleNewsUrl(urlStr, 0)
			if err != nil {
				fmt.Printf("Error decoding %s: %v\n", urlStr, err)
			} else {
				fmt.Printf("Decoded URL: %s\n", decoded)
			}
		}(u)
	}

	wg.Wait()
}
