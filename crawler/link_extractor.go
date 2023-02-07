package crawler

import (
	"context"
	"net/url"
	"regexp"

	"github.com/mycok/uSearch/pipeline"
)

// Static and compile-time check to ensure linkExtractor implements
// pipeline.Processor interface.
var _ pipeline.Processor = (*linkExtractor)(nil)

var (
	// Locate the <base href="xxx"> tag and return the value of the href attribute.
	baseHrefRegex = regexp.MustCompile(`(?i)<base.*?href\s*?=\s*?"(.*?)\s*?"`)
	// Locate the <a href="xxx"> tag and return the value of the href attribute.
	findLinkRegex = regexp.MustCompile(`(?i)<a.*?href\s*?=\s*?"\s*?(.*?)\s*?".*?>`)
	// Locate the <a href="xxx" rel="nofollow"> tag and return the value of the
	// rel attribute. This regex is used to isolate URLs that should be ignored
	// when calculating the page-rank score of a page.
	noFollowRegex = regexp.MustCompile(`(?i)rel\s*?=\s*?"?nofollow"?`)
)

// linkExtractor scans the body of each retrieved HTML document and extracts the
// all of the links embedded in it.
type linkExtractor struct {
	netDetector PrivateNetworkDetector
}

func newLinkExtractor(netDetector PrivateNetworkDetector) *linkExtractor {
	return &linkExtractor{netDetector}
}

// Process parses the [crawlerPayload.RawContent] data to extract absolute,
// relative, url with a network path reference, and no-follow links, appends
// them to [crawlerPayload.Links] and [crawlerPayload.NoFollowLinks] fields.
func (p *linkExtractor) Process(
	ctx context.Context, payload pipeline.Payload,
) (pipeline.Payload, error) {

	cPayload, ok := payload.(*crawlerPayload)
	if !ok {
		return nil, nil
	}

	// Generate a fully qualified url to use as a base URL for all relative
	// urls.
	relativeTo, err := url.Parse(cPayload.URL)
	if err != nil {
		return nil, err
	}

	content := cPayload.RawContent.String()

	// Search content for a <base href=""> tag and resolve it to an abs URL.
	// If the page content contains a base URL embedded in the <base href=xxx>
	//  tag, we should resolve it and use that as our base URL or relativeTo URL.
	baseMatches := baseHrefRegex.FindStringSubmatch(content)
	// Check if there is a match for a <base href=""> tag.
	// Note: len(baseMatches) will always return 2 or nil even when no submatch
	// match is found. this is because an empty string is always returned as a
	// place-holder. checks for an empty string are handled by the
	// resolveToAbsoluteURL() helper function.
	if len(baseMatches) == 2 {
		baseURL := resolveToAbsoluteURL(
			relativeTo, checkAndAddTrailingSlash(baseMatches[1]),
		)

		// If the page contains a base URL embedded in a <base href="xxx" tag,
		// then we use that as the base URL for all urls found in <a href="">
		// tags in the raw link content.
		if baseURL != nil {
			relativeTo = baseURL
		}
	}

	// Search for urls from the document data, if found, resolve them and
	// add them to the payload.
	seenMap := map[string]struct{}{}
	for _, match := range findLinkRegex.FindAllStringSubmatch(content, -1) {
		parsedURL := resolveToAbsoluteURL(relativeTo, match[1])
		if !p.shouldRetainURL(relativeTo.Hostname(), parsedURL) {
			continue
		}

		// Truncate / remove html anchors.
		// ie, in this ["https://example.com/index.html#foo"] url, the [#foo]
		// is dropped.
		parsedURL.Fragment = ""

		parsedURLString := parsedURL.String()

		// Skip URLs that point to files that don't contain html content.
		if exclusionRegex.MatchString(parsedURLString) {
			continue
		}

		// Check for duplicates.
		if _, exists := seenMap[parsedURLString]; exists {
			continue
		}

		// Add url to the seen map
		seenMap[parsedURLString] = struct{}{}

		// Check if the matched <a href=""> tag also contains a [rel="nofollow"]
		// attribute.
		if noFollowRegex.MatchString(match[0]) {
			cPayload.NoFollowLinks = append(cPayload.NoFollowLinks, parsedURLString)
		} else {
			cPayload.Links = append(cPayload.Links, parsedURLString)
		}
	}

	return cPayload, nil
}

func (p *linkExtractor) shouldRetainURL(srcHost string, url *url.URL) bool {
	// Skip links that could not be resolved.
	if url == nil {
		return false
	}

	// Skip links with non HTTP(S) schemes.
	if url.Scheme != "http" && url.Scheme != "https" {
		return false
	}

	// Keep relative links to the same host. No need for a private link check
	// is it would have already been done by the link fetching stage.
	if srcHost == url.Hostname() {
		return true
	}

	// Skip links that resolve to private networks.
	isPrivate, err := p.netDetector.IsNetworkPrivate(url.Hostname())
	if err != nil || isPrivate {
		return false
	}

	return true
}

func checkAndAddTrailingSlash(s string) string {
	if s[len(s)-1] != '/' {
		return s + "/"
	}

	return s
}

// resolveToAbsoluteURL expands target into an absolute URL using the following
//  rules:
// 		- targets starting with '//' are treated as absolute URLs that inherit
//   	the protocol / scheme from relativeTo.
// 		- all other targets are assumed to be relative to relTo.

// If the target URL cannot be parsed, an nil URL wil be returned.
func resolveToAbsoluteURL(relativeTo *url.URL, target string) *url.URL {
	targetLength := len(target)
	// Check if the target is an empty string.
	if targetLength == 0 {
		return nil
	}

	// Check for network path references. ["//example.com"]
	if targetLength >= 1 && target[0] == '/' {
		if targetLength >= 2 && target[1] == '/' {
			target = relativeTo.Scheme + ":" + target
		}
	}

	parsedURL, err := url.Parse(target)
	if err != nil {
		return nil
	}

	return relativeTo.ResolveReference(parsedURL)
}
