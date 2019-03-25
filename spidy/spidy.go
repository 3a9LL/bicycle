package spidy

import (
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"hash/fnv"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	ErrAlreadyVisited 	= errors.New("URL already visited")
	ErrNotBelongDomain 	= errors.New("URL doesnt belong to the domain")
	ErrCorruptURL		= errors.New("URL is corrupt")

	logerr = log.New(os.Stderr, "", 0)
)


type Spidy struct {
	targetURL			string
	domain				string
	storage				*Storage
	reURLMatch			*regexp.Regexp

	httpClient 			*HttpClient

	userAgent			string
	maxDepth			int
	maxBodySize			int

	wg					*sync.WaitGroup
}

func New(targetURL string, maxDepth, maxBodySize, reqPerSec int) (*Spidy, error) {
	var err error

	s := &Spidy{}
	s.domain, err = ExtractDomain(targetURL)
	if err != nil {
		return nil, err
	}
	// Domain + subdomain
	re_str := `^(?:https?:\/\/)?(?:[^@\/\n]+@)?(?:[a-zA-Z0-9]+\.)?` + strings.Replace(s.domain,".","\\.", -1) + `(?:(?::[0-9]+)?(\/[^\n]*)?)?$`
	s.reURLMatch = regexp.MustCompile(re_str)

	s.targetURL		= targetURL
	s.userAgent 	= "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:65.0) Gecko/20100101 Firefox/65.0"
	s.maxDepth 		= maxDepth
	s.maxBodySize 	= maxBodySize

	s.storage		= &Storage{}
	s.storage.Init()
	s.httpClient	= NewHttpClient(reqPerSec)

	s.wg 		= &sync.WaitGroup{}
	return s, nil
}

func (s *Spidy) Run() <-chan string {
	outChan := make(chan string, 1024*1024)

	go func() {
		defer close(outChan)

		s.Visit(s.targetURL, outChan)
		s.Visit("https://"	+ s.domain, outChan)
		s.Visit("http://"	+ s.domain, outChan)

		u, _ := url.Parse(s.targetURL)
		host := u.Host
		if host != s.domain {
			s.Visit("https://"	+ u.Host, outChan)
			s.Visit("http://"	+ u.Host, outChan)
		}

		s.wg.Wait()
	}()

	return outChan
}


// Private methods
func (s *Spidy) CheckURL(url string) error {
	// Domain + subdomain
	if !s.reURLMatch.Match([]byte(url)) {
		return ErrNotBelongDomain
	}

	// Is already visited?
	h := fnv.New64a()
	h.Write([]byte(url))
	uHash := h.Sum64()
	visited, err := s.storage.IsVisited(uHash)
	if err != nil {
		return err
	}
	if visited {
		return ErrAlreadyVisited
	}
	return s.storage.Visited(uHash)
}

func (s *Spidy) Visit_(url string, outChan chan<- string, depth int) error {
	if depth > s.maxDepth {
		return nil
	}
	if err := s.CheckURL(url); err != nil {
		return err
	}

	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Set("User-Agent", 	s.userAgent)
	request.Header.Set("Accept", 		"*/*")
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		outChan <- url
		resp, err := s.httpClient.Do(request, s.maxBodySize)
		if err != nil {
			logerr.Println(err)
			return
		}
		resp.FixCharset()

		doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(resp.Body))
		doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
			href, _ := sel.Attr("href")
			href = s.AbsoluteURL(url, href)
			s.Visit_(href, outChan, depth+1)
		})
	}()
	return nil
}

func (s *Spidy) Visit(url string, outChan chan<- string) error {
	return s.Visit_(url, outChan, 1)
}


func (s *Spidy) AbsoluteURL(srcUrl, relUrl string) string {
	if strings.HasPrefix(relUrl, "#") {
		return ""
	}
	base, err := url.Parse(srcUrl)
	if err != nil {
		return ""
	}
	absURL, err := base.Parse(relUrl)
	if err != nil {
		return ""
	}
	absURL.Fragment = ""
	if absURL.Scheme == "//" {
		absURL.Scheme = base.Scheme
	}
	return absURL.String()
}

// ststic
func ExtractDomain(uri string) (string, error){
	re := regexp.MustCompile(`^(?:https?:\/\/)?(?:[^@\/\n]+@)?(?:www\.)?([^:\/\n]+)`)
	if !re.MatchString(uri){
		return "", ErrCorruptURL
	}
	dom := re.FindStringSubmatch(uri)[1]
	return dom, nil
}