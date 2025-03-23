package main

import (
	"context"
	"errors"
	"flag"
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

func main() {
	if err := Main(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var errUsage = errors.New("Usage: ollamasearch <query>")

func Main() error {
	flag.Parse()
	if flag.NArg() == 0 {
		return errUsage
	}

	p := url.Values{}
	args := slices.Clone(flag.Args())
	for i, arg := range flag.Args() {
		c, ok := strings.CutPrefix(arg, "has:")
		if ok {
			args[i] = "" // do not include in query
			p.Add("c", c)
		}
	}

	p.Add("q", strings.Join(args, " "))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// example: https://ollama.com/search?c=tools&q=smol
	urlStr := (&url.URL{
		Scheme:   "https",
		Host:     "ollama.com",
		Path:     "/search",
		RawQuery: p.Encode(),
	}).String()

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "OllamaSearch/0.1")

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

// example:

//     <li x-test-model class="flex items-baseline border-b border-neutral-200 py-6">
//       <a href="/library/granite3.2-vision" class="group w-full">
//         <div class="flex flex-col mb-1" title="granite3.2-vision">
//           <h2 class="truncate text-xl font-medium underline-offset-2 group-hover:underline md:text-2xl">
//             <span x-test-search-response-title>granite3.2-vision</span>
//           </h2>
//           <p class="max-w-lg break-words text-neutral-800 text-md">A compact and efficient vision-language model, specifically designed for visual document unde
// rstanding, enabling automated content extraction from tables, charts, infographics, plots, diagrams, and more.</p>
//         </div>
//         <div class="flex flex-col">
//           <div class="flex flex-wrap space-x-2">
//
//             <span x-test-capability class="inline-flex my-1 items-center rounded-md bg-indigo-50 px-2 py-[2px] text-xs font-medium text-indigo-600 sm:text-[13px
// ]">vision</span>
//
//
//
//               <span x-test-capability class="inline-flex my-1 items-center rounded-md bg-indigo-50 px-2 py-[2px] text-xs font-medium text-indigo-600 sm:text-[13
// px]">tools</span>
//
//
//               <span x-test-size class="inline-flex my-1 items-center rounded-md bg-[#ddf4ff] px-2 py-[2px] text-xs font-medium text-blue-600 sm:text-[13px]">2b<
// /span>
//
//           </div>
//           <p class="my-1 flex space-x-5 text-[13px] font-medium text-neutral-500">
//
//               <span class="flex items-center">
//                 <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="mr-1.5 h-[14px] w-[14px]
//  sm:h-4 sm:w-4">
//                   <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0
// 0L7.5 12m4.5 4.5V3"></path>
//                 </svg>
//                 <span x-test-pull-count>27.7K</span>
//                 <span class="hidden sm:flex">&nbsp;Pulls</span>
//               </span>
//
//
//               <span class="flex items-center">
//                 <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="mr-1.5 h-[14px] w-[14px]
//  sm:h-4 sm:w-4">
//                   <path stroke-linecap="round" stroke-linejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.69
// 9 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z" />
//                   <path stroke-linecap="round" stroke-linejoin="round" d="M6 6h.008v.008H6V6z" />
//                 </svg>
//                 <span x-test-tag-count>5</span>
//                 <span class="hidden sm:flex">&nbsp;Tags</span>
//               </span>
//
//
//               <span class="flex items-center" title="Feb 27, 2025 7:26 PM UTC">
//                 <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="mr-1.5 h-[14px] w-[14px]
//  sm:h-4 sm:w-4">
//                   <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"></path>
//                 </svg>
//                 <span class="hidden sm:flex">Updated&nbsp;</span>
//                 <span x-test-updated>3 weeks ago</span>
//               </span>
//
//           </p>
//         </div>
//       </a>
//     </li>
