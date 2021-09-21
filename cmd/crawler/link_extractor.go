package crawler

import (
	"context"
	"net/url"
	"regexp"

	"github.com/mycok/uSearch/internal/pipeline"
)

// Compile-time check for ensuring linkExtractor implements pipeline.Processor.
var _ pipeline.Processor = (*linkExtractor)(nil)

var (
	baseHrefRegex = regexp.MustCompile(`(?i)<base.*?href\s*?=\s*?"(.*?)\s*?"`)
	findLinkRegex = regexp.MustCompile(`(?i)<a.*?href\s*?=\s*?"\s*?(.*?)\s*?".*?>`)
	noFollowRegex = regexp.MustCompile(`(?i)rel\s*?=\s*?"?nofollow"?`)
)

// linkExtractor scans the body of each retrieved HTML document and extracts the unique set of
// links contained within it.
type linkExtractor struct {
	netDetector PrivateNetworkDetector // This object is used to filter out links that belong to private networks.
}

func newLinkExtractor(netDetector PrivateNetworkDetector) *linkExtractor {
	return &linkExtractor{netDetector: netDetector}
}

func (le *linkExtractor) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)
	// Generate a fully qualified link to use as a base URL for all relative links.
	relativeTo, err := url.Parse(payload.URL)
	if err != nil {
		return nil, err
	}

	content := payload.RawContent.String()

	// Search page content for a <base> tag and resolve it to an abs URL.
	// If the page content contains a base URL embedded in the <base href=xxx> tag, we should resolve it
	// and use that as our base URL or relativeTo URL.
	if baseMatch := baseHrefRegex.FindStringSubmatch(content); len(baseMatch) == 2 {
		if base := resolveToAbsoluteURL(relativeTo, checkAndAddtrailingSlash(baseMatch[1])); base != nil {
			relativeTo = base
		}
	}

	// Find the unique set of links from the document, resolve them and
	// add them to the payload.
	seenMap := make(map[string]struct{})
	for _, match := range findLinkRegex.FindAllStringSubmatch(content, -1) {
		link := resolveToAbsoluteURL(relativeTo, match[1])
		if !le.shouldRetainUrlLink(relativeTo.Hostname(), link) {
			continue
		}

		// Truncate anchors and drop duplicates.
		link.Fragment = ""
		linkStr := link.String()
		if _, seen := seenMap[linkStr]; seen {
			continue
		}

		// Skip URLs that point to files that cannot contain html content.
		if exclusionRegex.MatchString(linkStr) {
			continue
		}

		seenMap[linkStr] = struct{}{}
		if noFollowRegex.MatchString(match[0]) {
			payload.NoFollowLinks = append(payload.NoFollowLinks, linkStr)
		} else {
			payload.Links = append(payload.Links, linkStr)
		}
	}

	return payload, nil
}

func (le *linkExtractor) shouldRetainUrlLink(srcHost string, urlLink *url.URL) bool {
	// Skip links that could not be resolved.
	if urlLink == nil {
		return false
	}

	// Skip links with non HTTP(S) schemes.
	if urlLink.Scheme != "http" && urlLink.Scheme != "https" {
		return false
	}

	// Skip links that resolve to private networks.
	if isPrivate, err := le.netDetector.IsPrivate(urlLink.Host); err != nil || isPrivate {
		return false
	}

	// Keep links to the same host.
	if urlLink.Hostname() == srcHost {
		return true
	}

	return true
}

func checkAndAddtrailingSlash(s string) string {
	if s[len(s) - 1] != '/' {
		return s + "/"
	}

	return s
}

// resolveURL expands target into an absolute URL using the following rules:
// - targets starting with '//' are treated as absolute URLs that inherit the
//   protocol from relativeTo [base / parent URL].
// - targets starting with '/' are absolute URLs that are appended to the host
//   from relativeTo [base / parent URL].
// - all other targets are assumed to be relative to relativeTo [base / parent URL].
//
// If the target URL cannot be parsed, an nil URL will be returned.
func resolveToAbsoluteURL(relativeTo *url.URL, target string) *url.URL {
	tLen := len(target)
	if tLen == 0 { 
		return nil
	}

	if tLen >= 1 && target[0] == '/' {
		if tLen >= 2 && target[1] == '/' {
			target = relativeTo.Scheme + ":" + target
		}
	}

	if targetURL, err := url.Parse(target); err == nil {
		return relativeTo.ResolveReference(targetURL)
	}

	return nil
}
