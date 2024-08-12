package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Parser struct {
	URLs              map[string]struct{}
	excludedFileTypes []string
	reHTML            *regexp.Regexp
	reURL             *regexp.Regexp
	file              *os.File
	client            *http.Client
	count             int32
}

// init parser
func (p *Parser) Init() error {
	// set file for pages
	f, err := os.Create("pages.txt")
	if err != nil {
		return err
	}
	p.file = f

	// init regexps
	if err := p.initRegularAndExcludes(); err != nil {
		return err
	}

	// create http client with timeout
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	p.client = &client

	// init map
	p.URLs = make(map[string]struct{})

	return nil
}

// init parser regular expression and excludes (HTML and URL)
func (p *Parser) initRegularAndExcludes() error {
	// compile the HTML regular expression
	patternHTML := `href="(https://monzo\.com/[^\s'"]+|/[^"\s']+)"`
	reHTML, err := regexp.Compile(patternHTML)
	if err != nil {
		return err
	}
	p.reHTML = reHTML

	// compile the URL extra regular expression
	p.excludedFileTypes = []string{
		".html", ".htm", ".css", ".js", ".json", ".xml", ".pdf", ".txt",
		".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".bmp", ".tiff", ".tif",
		".mp3", ".wav", ".ogg", ".mp4", ".webm", ".avi", ".mov", ".mkv",
		".zip", ".rar", ".7z", ".tar", ".gz",
		".ico", ".eot", ".woff", ".ttf", ".otf",
	}
	patternURL := `(?i)\b(?:` + joinExtensions(p.excludedFileTypes) + `)\b$`
	reURL, err := regexp.Compile(patternURL)
	if err != nil {
		return err
	}
	p.reURL = reURL

	return nil
}

// Entrypoint
func (p *Parser) Parse(url string) {
	// log.Printf("[%v]: %s", p.count, url)
	url = strings.TrimRight(url, "/")
	if _, ok := p.URLs[url]; ok {
		return
	}
	p.URLs[url] = struct{}{}

	// get html
	html, err := p.getHTMLString(url)
	if err != nil {
		log.Printf("[FAIL] %s: %s\n", url, err.Error())
		return
	}

	// find all matches in the html
	matches := p.reHTML.FindAllStringSubmatch(html, -1)

	// check links
	for _, match := range matches {
		tmp := strings.TrimRight(match[1], "/")
		if p.isExcluded(tmp) {
			continue
		}
		if strings.HasPrefix(tmp, "/") {
			tmp = baseURL + tmp
		}
		_, ok := p.URLs[tmp]
		if !ok {
			p.count++
			p.file.WriteString(tmp + "\n")
			p.Parse(tmp)
		}
	}

}

// make GET request and return HTML string
func (p *Parser) getHTMLString(url string) (string, error) {
	resp, err := p.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// check URL
func (p *Parser) isExcluded(url string) bool {
	for _, fileType := range p.excludedFileTypes {
		if strings.HasSuffix(url, fileType) {
			return true
		}
	}
	return strings.Contains(url, "-deeplinks") || // got these dead links from pages
		strings.Contains(url, "/i/") || /* remove duplicates (I didn't check all the '/i/...' links,
		but generally, they redirect to existing links) */
		strings.Contains(url, "/undefined/") || // got infinity pages count with undefined
		strings.ContainsAny(url, "#[]") // remove duplicate pages with different scrolls and any other unusual pages.
}

// [Deprecated] does the same as isExcluded, has lower speed
func (p *Parser) isExcludedByRegexp(url string) bool {
	match := p.reURL.FindStringSubmatch(url)
	return len(match) >= 1 ||
		strings.Contains(url, "-deeplinks") || // got these dead links from pages
		strings.Contains(url, "/i/") || /* remove duplicates (I didn't check all the '/i/...' links,
		but generally, they redirect to existing links) */
		strings.Contains(url, "/undefined/") || // got infinity pages count with undefined
		strings.ContainsAny(url, "#[]") // remove duplicate pages with different scrolls and any other unusual pages.
}

// close file
func (p *Parser) Close() error {
	return p.file.Close()
}

// helper function to join extensions into a regex pattern
func joinExtensions(extensions []string) string {
	var sb strings.Builder
	for i, ext := range extensions {
		if i > 0 {
			sb.WriteString("|")
		}
		sb.WriteString(regexp.QuoteMeta(ext))
	}
	return sb.String()
}
