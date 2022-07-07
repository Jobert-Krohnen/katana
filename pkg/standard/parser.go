package standard

import (
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/projectdiscovery/katana/pkg/navigation"
	"github.com/projectdiscovery/katana/pkg/utils"
)

// responseParserFunc is a function that parses the document returning
// new navigation items or requests for the crawler.
type responseParserFunc func(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest))

// responseParsers is a list of response parsers for the standard engine
var responseParsers = []responseParserFunc{
	// Header based parsers
	headerContentLocationParser,
	headerLinkParser,
	headerLocationParser,
	headerRefreshParser,

	// Body based parsers
	bodyATagParser,
	bodyEmbedTagParser,
	bodyFrameTagParser,
	bodyIframeTagParser,
	bodyInputSrcTagParser,
	bodyIsindexActionTagParser,
	bodyScriptSrcTagParser,
	bodyFormTagParser,
	bodyMetaContentTagParser,

	// Optional JS relative endpoints parsers
	scriptContentRegexParser,
	scriptJSFileRegexParser,
}

// parseResponse runs the response parsers on the navigation response
func parseResponse(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	for _, parser := range responseParsers {
		parser(resp, callback)
	}
}

// -------------------------------------------------------------------------
// Begin Header based parsers
// -------------------------------------------------------------------------

// headerContentLocationParser parsers Content-Location header from response
func headerContentLocationParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	header := resp.Resp.Header.Get("Content-Location")
	if header == "" {
		return
	}
	callback(navigation.NewNavigationRequestURL(header, "content-location", resp))
}

// headerLinkParser parsers Link header from response
func headerLinkParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	header := resp.Resp.Header.Get("Link")
	if header == "" {
		return
	}
	values := utils.ParseLinkTag(header)
	for _, value := range values {
		callback(navigation.NewNavigationRequestURL(value, "link", resp))
	}
}

// headerLocationParser parsers Location header from response
func headerLocationParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	header := resp.Resp.Header.Get("Location")
	if header == "" {
		return
	}
	callback(navigation.NewNavigationRequestURL(header, "location", resp))
}

// headerRefreshParser parsers Refresh header from response
func headerRefreshParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	header := resp.Resp.Header.Get("Refresh")
	if header == "" {
		return
	}
	values := utils.ParseRefreshTag(header)
	if values == "" {
		return
	}
	callback(navigation.NewNavigationRequestURL(values, "refresh", resp))
}

// -------------------------------------------------------------------------
// Begin Body based parsers
// -------------------------------------------------------------------------

// bodyATagParser parses A tag from response
func bodyATagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("a").Each(func(i int, item *goquery.Selection) {
		href, ok := item.Attr("href")
		if ok && href != "" {
			callback(navigation.NewNavigationRequestURL(href, "a", resp))
		}
		ping, ok := item.Attr("ping")
		if ok && ping != "" {
			callback(navigation.NewNavigationRequestURL(ping, "a", resp))
		}
	})
}

// bodyEmbedTagParser parses Embed tag from response
func bodyEmbedTagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("embed[src]").Each(func(i int, item *goquery.Selection) {
		src, ok := item.Attr("src")
		if ok && src != "" {
			callback(navigation.NewNavigationRequestURL(src, "embed", resp))
		}
	})
}

// bodyFrameTagParser parses frame tag from response
func bodyFrameTagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("frame[src]").Each(func(i int, item *goquery.Selection) {
		src, ok := item.Attr("src")
		if ok && src != "" {
			callback(navigation.NewNavigationRequestURL(src, "frame", resp))
		}
	})
}

// bodyIframeTagParser parses iframe tag from response
func bodyIframeTagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("iframe[src]").Each(func(i int, item *goquery.Selection) {
		src, ok := item.Attr("src")
		if ok && src != "" {
			callback(navigation.NewNavigationRequestURL(src, "iframe", resp))
		}
	})
}

// bodyInputSrcTagParser parses input image src tag from response
func bodyInputSrcTagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("input[type='image']").Each(func(i int, item *goquery.Selection) {
		src, ok := item.Attr("src")
		if ok && src != "" {
			callback(navigation.NewNavigationRequestURL(src, "input", resp))
		}
	})
}

// bodyIsindexActionTagParser parses isindex action tag from response
func bodyIsindexActionTagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("isindex[action]").Each(func(i int, item *goquery.Selection) {
		src, ok := item.Attr("action")
		if ok && src != "" {
			callback(navigation.NewNavigationRequestURL(src, "isindex", resp))
		}
	})
}

// bodyScriptSrcTagParser parses script src tag from response
func bodyScriptSrcTagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("script[src]").Each(func(i int, item *goquery.Selection) {
		src, ok := item.Attr("src")
		if ok && src != "" {
			callback(navigation.NewNavigationRequestURL(src, "script", resp))
		}
	})
}

// bodyButtonFormactionTagParser parses button formaction tag from response
func bodyButtonFormactionTagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("button[formaction]").Each(func(i int, item *goquery.Selection) {
		src, ok := item.Attr("formaction")
		if ok && src != "" {
			callback(navigation.NewNavigationRequestURL(src, "button", resp))
		}
	})
}

// bodyFormTagParser parses forms from response
func bodyFormTagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("form[action]").Each(func(i int, item *goquery.Selection) {
		href, ok := item.Attr("action")
		if !ok {
			return
		}
		encType, ok := item.Attr("enctype")
		if !ok || encType == "" {
			encType = "application/x-www-form-urlencoded"
		}

		method, _ := item.Attr("method")
		if method == "" {
			method = "GET"
		}
		method = strings.ToUpper(method)

		actionURL := resp.AbsoluteURL(href)
		if actionURL == "" {
			return
		}

		isMultipartForm := strings.HasPrefix(encType, "multipart/")

		queryValuesWriter := make(url.Values)
		var sb strings.Builder
		var multipartWriter *multipart.Writer

		if isMultipartForm {
			multipartWriter = multipart.NewWriter(&sb)
		}

		// Get the form field suggestions for all inputs
		formInputs := []utils.FormInput{}
		item.Find("input").Each(func(index int, item *goquery.Selection) {
			if len(item.Nodes) == 0 {
				return
			}
			formInputs = append(formInputs, utils.ConvertGoquerySelectionToFormInput(item))
		})

		dataMap := utils.FormInputFillSuggestions(formInputs, utils.DefaultFormFillData)
		for key, value := range dataMap {
			if key == "" || value == "" {
				continue
			}
			if isMultipartForm {
				_ = multipartWriter.WriteField(key, value)
			} else {
				queryValuesWriter.Set(key, value)
			}
		}

		// Guess content-type
		var contentType string
		if multipartWriter != nil {
			multipartWriter.Close()
			contentType = multipartWriter.FormDataContentType()
		} else {
			contentType = encType
		}

		req := navigation.NavigationRequest{
			Method: method,
			URL:    actionURL,
			Depth:  resp.Depth,
			Source: "form",
		}
		if multipartWriter != nil {
			req.Body = sb.String()
		} else {
			req.Body = queryValuesWriter.Encode()
		}
		switch method {
		case "GET":
			value := queryValuesWriter.Encode()
			sb.Reset()
			sb.WriteString(req.URL)
			sb.WriteString("?")
			sb.WriteString(value)
			req.URL = sb.String()
		case "POST":
			req.Headers = make(map[string]string)
			req.Headers["Content-Type"] = contentType
		}
		callback(req)
	})
}

// bodyMetaContentTagParser parses meta content tag from response
func bodyMetaContentTagParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("meta[http-equiv='refresh']").Each(func(i int, item *goquery.Selection) {
		header, ok := item.Attr("content")
		if !ok {
			return
		}
		values := utils.ParseRefreshTag(header)
		if values == "" {
			return
		}
		callback(navigation.NewNavigationRequestURL(values, "meta", resp))
	})
}

// -------------------------------------------------------------------------
// Begin JS Regex based parsers
// -------------------------------------------------------------------------

// scriptContentRegexParser parses script content endpoints from response
func scriptContentRegexParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	resp.Reader.Find("script").Each(func(i int, item *goquery.Selection) {
		if !resp.Options.Options.ScrapeJSResponses { // do not process if disabled
			return
		}
		text := item.Text()
		if text == "" {
			return
		}
		endpoints := utils.ExtractRelativeEndpoints(text)
		for _, item := range endpoints {
			callback(navigation.NewNavigationRequestURL(item, "script-content", resp))
		}
	})
}

// scriptJSFileRegexParser parses relative endpoints from js file pages
func scriptJSFileRegexParser(resp navigation.NavigationResponse, callback func(navigation.NavigationRequest)) {
	if !resp.Options.Options.ScrapeJSResponses { // do not process if disabled
		return
	}

	// Only process javascript file based on path or content type
	contentType := resp.Resp.Header.Get("Content-Type")
	if !(strings.HasSuffix(resp.Resp.Request.URL.Path, ".js") || strings.Contains(contentType, "/javascript")) {
		return
	}

	endpoints := utils.ExtractRelativeEndpoints(string(resp.Body))
	for _, item := range endpoints {
		callback(navigation.NewNavigationRequestURL(item, "js-file", resp))
	}
}