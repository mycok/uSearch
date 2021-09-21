package crawler

import (
	"context"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/mycok/uSearch/internal/pipeline"
)

// Compile-time check for ensuring linkFetcher implements pipeline.Processor.
var _ pipeline.Processor = (*linkFetcher)(nil)

var exclusionRegex = regexp.MustCompile(`(?i)\.(?:jpg|jpeg|png|gif|ico|css|js)$`)

// linkFetcher serves as the first stage of the crawler pipeline. it operates
// on payload values emitted by the input source and attempts to retrieve the contents
// of each link by performing an http GET request.
// The successful retrieved link contents are stored within the payload's [RawContent] field
// and made available to the following stage of the pipeline.
type linkFetcher struct {
	urlGetter   URLGetter              // This object provides the required implementation for performing HTTP GET requests.
	netDetector PrivateNetworkDetector // This object is used to filter out links that belong to private networks.
}

// NewLinkFetcher initializes and returns a new instance of linkFetcher.
func newLinkFetcher(urlGetter URLGetter, netDetector PrivateNetworkDetector) *linkFetcher {
	return &linkFetcher{
		urlGetter:   urlGetter,
		netDetector: netDetector,
	}
}

func (lf *linkFetcher) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	// Type assert / cast the received payload to the crawlerPayload type.
	payload := p.(*crawlerPayload)

	// Skip URLs that point to files that cannot contain html content.
	if exclusionRegex.MatchString(payload.URL) {
		return nil, nil
	}

	// Never crawl links in private networks (e.g. link-local addresses).
	// it is a security risk!.
	if isPrivate, err := lf.isPrivate(payload.URL); err != nil || isPrivate {
		return nil, nil
	}

	// Perform an HTTP GET request for the provided payload url.
	resp, err := lf.urlGetter.Get(payload.URL)
	if err != nil {
		return nil, nil
	}

	// Skip payloads with invalid response status codes.
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, nil
	}

	// Skip for non html payloads.
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "html") {
		return nil, nil
	}

	_, err = io.Copy(&payload.RawContent, resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (lf *linkFetcher) isPrivate(urlStr string) (bool, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false, err
	}

	return lf.netDetector.IsPrivate(u.Hostname())

}
