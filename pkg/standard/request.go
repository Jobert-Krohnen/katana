package standard

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"github.com/projectdiscovery/katana/pkg/navigation"
	"github.com/projectdiscovery/katana/pkg/utils"
	"github.com/projectdiscovery/retryablehttp-go"
)

// makeRequest makes a request to a URL returning a response interface.
func (c *Crawler) makeRequest(request navigation.NavigationRequest) (navigation.NavigationResponse, error) {
	response := navigation.NavigationResponse{
		Depth:   request.Depth + 1,
		Options: c.options,
	}
	httpReq, err := http.NewRequest(request.Method, request.URL, nil)
	if err != nil {
		return response, err
	}
	if request.Body != "" {
		httpReq.Body = ioutil.NopCloser(strings.NewReader(request.Body))
	}
	req, err := retryablehttp.FromRequest(httpReq)
	if err != nil {
		return response, err
	}
	req.Header.Set("User-Agent", utils.WebUserAgent())

	// Set the headers for the request.
	for k, v := range request.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range c.options.Options.CustomHeaders.AsMap() {
		req.Header.Set(k, v.(string))
	}

	resp, err := c.httpclient.Do(req)
	if resp != nil {
		defer func() {
			_, _ = io.CopyN(ioutil.Discard, resp.Body, 8*1024)
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return response, err
	}
	if resp.StatusCode == 404 {
		return response, nil
	}
	limitReader := io.LimitReader(resp.Body, int64(c.options.Options.BodyReadSize))
	data, err := ioutil.ReadAll(limitReader)
	if err != nil {
		return response, err
	}
	response.Body = data
	response.Resp = resp
	response.Reader, err = goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return response, errors.Wrap(err, "could not make document from reader")
	}
	return response, nil
}