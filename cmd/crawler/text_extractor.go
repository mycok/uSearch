package crawler

import (
	"context"
	"html"
	"regexp"
	"strings"
	"sync"

	"github.com/mycok/uSearch/internal/pipeline"

	htmlPaser "github.com/microcosm-cc/bluemonday"
)

// Compile-time check for ensuring textExtractor implements pipeline.Processor.
var _ pipeline.Processor = (*textExtractor)(nil)

var (
	titleRegex         = regexp.MustCompile(`(?i)<title.*?>(.*?)</title>`)
	repeatedSpaceRegex = regexp.MustCompile(`\s+`)
)

// textExtractor extracts an index-friendly, text-only version of the web page contents.
type textExtractor struct {
	// policyPool contains a re-usable pool of the HTML parser instance.
	policyPool sync.Pool
}

func newTextExtractor() *textExtractor {
	return &textExtractor{
		policyPool: sync.Pool{
			New: func() interface{} {
				return htmlPaser.StrictPolicy()
			},
		},
	}
}

// Process parses the HTML document and stripping off all HTML tags and white spaces then updates the
// [crawlerPayload.Title, crawlerPayload.TextContext] with the resulting text.
func (te *textExtractor) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)
	policy := te.policyPool.Get().(*htmlPaser.Policy)

	if titleMatch := titleRegex.FindStringSubmatch(payload.RawContent.String()); len(titleMatch) == 2 {
		payload.Title = strings.TrimSpace(html.UnescapeString(repeatedSpaceRegex.ReplaceAllString(
			policy.Sanitize(titleMatch[1]), "",
		)))
	}

	payload.TextContent = strings.TrimSpace(html.UnescapeString(repeatedSpaceRegex.ReplaceAllString(
		policy.SanitizeReader(&payload.RawContent).String(), "",
	)))

	te.policyPool.Put(policy)

	return payload, nil
}
