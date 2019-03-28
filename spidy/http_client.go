package spidy

import (
	"bytes"
	"compress/gzip"
	"github.com/saintfish/chardet"
	"golang.org/x/net/html/charset"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

type Response struct {
	StatusCode		int
	Body			[]byte
	Headers			*http.Header
}

func EncodeBytes(b []byte, contentType string) ([]byte, error) {
	r, err := charset.NewReader(bytes.NewReader(b), contentType)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func (r *Response) FixCharset() error {
	contentType := strings.ToLower(r.Headers.Get("Content-Type"))
	if !strings.Contains(contentType, "charset") {
		d := chardet.NewTextDetector()
		r, err := d.DetectBest(r.Body)
		if err != nil {
			return err
		}
		contentType = "text/plain; charset=" + r.Charset
	}
	if strings.Contains(contentType, "utf-8") || strings.Contains(contentType, "utf8") {
		return nil
	}
	tmpBody, err := EncodeBytes(r.Body, contentType)
	if err != nil {
		return err
	}
	r.Body = tmpBody
	return nil
}

type HttpClient struct {
	limiter 	*Limiter

	client		*http.Client
	lock   		*sync.RWMutex
}

func NewHttpClient(reqPerSec int) *HttpClient {
	c := new(HttpClient)
	c.client 				= &http.Client{}
	c.client.CheckRedirect 	= func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	c.limiter				= NewLimiter(reqPerSec)
	return  c
}

func (c *HttpClient) Do(request *http.Request, bodySize int) (*Response, error) {
	c.limiter.Take()
	res, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	if res.Request != nil {
		*request = *res.Request
	}

	var bodyReader io.Reader = res.Body
	if bodySize > 0 {
		bodyReader = io.LimitReader(bodyReader, int64(bodySize))
	}
	if !res.Uncompressed && res.Header.Get("Content-Encoding") == "gzip" {
		bodyReader, err = gzip.NewReader(bodyReader)
		if err != nil {
			return nil, err
		}
	}
	body, err := ioutil.ReadAll(bodyReader)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	return &Response{
		StatusCode: res.StatusCode,
		Body:       body,
		Headers:    &res.Header,
	}, nil
}
