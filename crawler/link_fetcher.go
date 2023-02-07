package crawler

import (
	"context"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/mycok/uSearch/pipeline"
)

var (
	// Static and compile-time check to ensure linkFetcher implements
	// pipeline.Processor interface.
	_ pipeline.Processor = (*linkFetcher)(nil)

	// Locate links that point to web pages that don't serve html content.
	exclusionRegex = regexp.MustCompile(`(?i)\.(?:jpg|jpeg|png|gif|ico|css|js)$`)
)

// linkFetcher serves as the first stage processor of the crawler pipeline. it
// processes payload values emitted by the input source and attempts to retrieve
// the contents of each link by performing an http GET request.
// The successful retrieved link contents are stored within the payload's
// [RawContent] field and made available to the following stage(s) of the
// pipeline.
type linkFetcher struct {
	urlGetter   URLGetter
	netDetector PrivateNetworkDetector
}

func newLinkFetcher(
	urlGetter URLGetter, netDetector PrivateNetworkDetector,
) *linkFetcher {

	return &linkFetcher{
		urlGetter:   urlGetter,
		netDetector: netDetector,
	}
}

// Process may transform the payload and return the transformed payload that
// may be forwarded to the next stage in the pipeline. Processors may also
// opt to prevent the payload from reaching the next stage of the pipeline by
// returning a nil payload value instead. for example, if the payload is
// malformed or doesn't meet the required conditions.
func (p *linkFetcher) Process(
	ctx context.Context, payload pipeline.Payload,
) (pipeline.Payload, error) {

	cPayload, ok := payload.(*crawlerPayload)
	if !ok {
		return nil, nil
	}

	// Filter out (drop) links that point to files that don't contain html
	//  content.
	if exclusionRegex.MatchString(cPayload.URL) {
		// Return a nil payload to indicate to the caller that the payload has
		// been discarded / dropped since it doesn't contain data of valid type.
		return nil, nil
	}

	// Filter out links to private networks, since crawling such links is a
	// security risk
	isPrivate, err := p.isNetworkPrivate(cPayload.URL)
	if err != nil || isPrivate {
		// Return a nil payload to indicate to the caller that the payload has
		// been discarded / dropped since it points to a private network or
		// it's an invalid address
		return nil, nil
	}

	// Perform an HTTP GET request for the provided payload url.
	resp, err := p.urlGetter.Get(cPayload.URL)
	if err != nil {
		// Return a nil payload to indicate to the caller that the payload has
		// been discarded / dropped because of an error from the resource server.
		return nil, nil
	}

	// Skip payloads with non success response status codes (only allow 2xx).
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, nil
	}

	// Skip for non html payloads.
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "html") {
		return nil, nil
	}

	_, err = io.Copy(&cPayload.RawContent, resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return cPayload, nil
}

func (p *linkFetcher) isNetworkPrivate(urlStr string) (bool, error) {
	url, err := url.Parse(urlStr)
	if err != nil {
		return false, err
	}

	return p.netDetector.IsNetworkPrivate(url.Hostname())
}
