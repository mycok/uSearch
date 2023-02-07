package crawler

import (
	"context"
	"html"
	"regexp"
	"strings"
	"sync"

	"github.com/microcosm-cc/bluemonday"

	"github.com/mycok/uSearch/pipeline"
)

// Static and compile-time check to ensure textExtractor implements
// pipeline.Processor interface.
var _ pipeline.Processor = (*textExtractor)(nil)

var (
	titleRegex         = regexp.MustCompile(`(?i)<title.*?>(.*?)</title>`)
	repeatedSpaceRegex = regexp.MustCompile(`\s+`)
)

type textExtractor struct {
	policyPool sync.Pool
}

func newTextExtractor() *textExtractor {
	return &textExtractor{
		policyPool: sync.Pool{
			New: func() interface{} {
				return bluemonday.StrictPolicy()
			},
		},
	}
}

// Process parses the payload's raw content, extracts and assigns the title
// field then strips the content of all HTML tags and unnecessary white spaces,
// and assigns the textContent field of the payload.
func (p *textExtractor) Process(
	ctx context.Context, payload pipeline.Payload,
) (pipeline.Payload, error) {

	cPayload, ok := payload.(*crawlerPayload)
	if !ok {
		return nil, nil
	}

	policy := p.policyPool.Get().(*bluemonday.Policy)

	titleMatch := titleRegex.FindStringSubmatch(cPayload.RawContent.String())
	// Note: len(titleMatch) always returns 2 or nil even when no submatch
	// match is found. this is because an empty string is always returned as a
	// place-holder.
	if len(titleMatch) == 2 {
		cleanTitle := repeatedSpaceRegex.ReplaceAllString(
			policy.Sanitize(titleMatch[1]), " ",
		)

		cPayload.Title = strings.TrimSpace(html.UnescapeString(cleanTitle))
	}

	cleanContent := repeatedSpaceRegex.ReplaceAllString(
		policy.SanitizeReader(&cPayload.RawContent).String(), " ",
	)

	cPayload.TextContent = strings.TrimSpace(html.UnescapeString(cleanContent))

	p.policyPool.Put(policy)

	return cPayload, nil
}
