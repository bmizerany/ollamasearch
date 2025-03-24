package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"golang.org/x/net/html"
)

var errUsage = errors.New(`
Usage: ollamasearch <query>

Use "has:" to filter by capability. For example:

	ollamasearch "has:tools has:vision gemma"

The query must be the first param, not spread across multiple params. Use
quotes if you need spaces.
`[1:])

var envDebug = os.Getenv("OLLAMASEARCHDEBUG") != ""

func vlogf(format string, args ...any) {
	if envDebug {
		fmt.Fprintf(os.Stderr, "DEBUG: "+format, args...)
	}
}

func main() {
	if err := Main(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Main() error {
	if len(os.Args) > 2 {
		return errUsage
	}
	var query string
	if len(os.Args) == 2 {
		query = os.Args[1]
	}

	var b strings.Builder
	p := url.Values{}
	for arg := range strings.FieldsSeq(query) {
		c, ok := strings.CutPrefix(arg, "has:")
		if ok {
			p.Add("c", c)
		} else {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(arg)
		}
	}
	p.Add("q", b.String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// example: https://ollama.com/search?c=tools&q=smol
	urlStr := (&url.URL{
		Scheme:   "https",
		Host:     "ollama.com",
		Path:     "/search",
		RawQuery: p.Encode(),
	}).String()

	vlogf("GET %s\n", urlStr)

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "OllamaSearch/0.1")
	req.Header.Set("Hx-Request", "true") // we only care about results, not the full layout

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	root, err := html.Parse(resp.Body)
	if err != nil {
		return err
	}

	// Find first model
	var model *html.Node
	for node := range root.Descendants() {
		if hasAttr(node, "x-test-model") {
			model = node
			break
		}
	}

	tw := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
	defer tw.Flush()

	for model != nil {
		var name string
		var caps []string
		var desc string
		for node := range model.Descendants() {
			if hasAttr(node, "x-test-search-response-title") {
				name = innerText(node)
			}
			if hasAttr(node, "x-test-capability") {
				caps = append(caps, innerText(node))
			}
			if desc == "" && node.Data == "p" { // no clear attr for description, so just look for the p
				desc = innerText(node)
			}
		}

		slices.Sort(caps)
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			name,
			strings.Join(caps, " + "),
			ellipsis(desc, 80),
		)

		for {
			model = model.NextSibling
			if model != nil && !hasAttr(model, "x-test-model") {
				continue
			}
			break
		}
	}
	return nil
}

func ellipsis(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func innerText(node *html.Node) string {
	if node.FirstChild == nil {
		return ""
	}
	return strings.TrimSpace(node.FirstChild.Data)
}

func hasAttr(node *html.Node, attr string) bool {
	for _, a := range node.Attr {
		if a.Key == attr {
			return true
		}
	}
	return false
}
