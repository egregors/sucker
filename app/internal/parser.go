package internal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

// New create a new Parser and pass pages urls into pages channel
func NewHtmlParser(pagesUrls, exts []string) *HtmlParser {
	p := new(HtmlParser)
	p.setExts(exts)
	p.pages = make(chan string)
	p.links = make(chan string)

	go func() {
		defer close(p.pages)
		for _, url := range pagesUrls {
			p.pages <- url
		}
	}()
	return p
}

// HtmlParser searching links for content files with required extensions
type HtmlParser struct {
	pages, links chan string
	exts         []string
}

// GetLinks starts async parsing and return chan with files urls
func (p *HtmlParser) GetLinks() []string {
	linksSet := map[string]bool{}
	for l := range p.getLinksChan() {
		linksSet[l] = true
	}
	var links []string
	for link := range linksSet {
		links = append(links, link)
	}
	sort.Strings(links)
	return links
}

func (p *HtmlParser) getLinksChan() chan string {
	// todo: pass context with timeout from outside
	go p.parseAll(nil)
	return p.links
}

func (p *HtmlParser) setExts(exts []string) {
	// todo: pass defaults from main somehow
	p.exts = exts
	if len(p.exts) == 0 {
		p.exts = []string{"webm", "mp4"}
	}
}

func (p *HtmlParser) parseAll(ctx context.Context) {
	defer close(p.links)

	if ctx == nil {
		ctx = context.Background()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case page, ok := <-p.pages:
			if !ok {
				return
			}

			err := p.parsePage(page)
			if err != nil {
				log.Printf("[ERROR] can't parse page: %v", err)
				continue
			}
		}
	}
}

func (p *HtmlParser) parsePage(page string) error {
	doc, err := getHtmlDocument(page)
	if err != nil {
		return errors.New(fmt.Sprintf("[ERROR] can't get page: %s becouse of: %v", page, err))
	}
	p.checkNode(doc, makeDomainName(page))
	return nil
}

func getHtmlDocument(pageUrl string) (*html.Node, error) {

	resp, err := makeGetRequest(pageUrl)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("[ERROR] can't parse page: %v", err))
	}
	_ = resp.Body.Close()

	return doc, nil
}

func makeGetRequest(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("[ERROR] can't get page: %v", err))
	}
	return resp, nil
}

func (p *HtmlParser) checkNode(n *html.Node, domain string) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" && p.isValidHref(a.Val) {
				p.links <- domain + a.Val
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		p.checkNode(c, domain)
	}
}

func (p *HtmlParser) isValidHref(href string) bool {
	for _, e := range p.exts {
		if strings.Contains(href, e) {
			return true
		}
	}
	return false
}

func makeDomainName(page string) string {
	domain := strings.Split(page, "/")
	return domain[0] + "//" + domain[1] + domain[2]
}
