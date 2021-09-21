package crawler

import (
	"context"

	check "gopkg.in/check.v1"
)

// Initialize and register an instance of TextExtractorTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(TextExtractorTestSuite))

type TextExtractorTestSuite struct{}

func (s *TextExtractorTestSuite) TestContentExtraction(c *check.C) {
	content := `<div>
					Some<span> content</span> rock &amp; roll
				</div>
				<buttton>Search</button>
				`
	assertExtractedText(c, content, "", `Some content rock & roll Search`)
}

func (s *TextExtractorTestSuite) TestContentAndTitleExtraction(c *check.C) {
	content := `<html>
					<head><title>test title</title></head>
					<body>
						<div>
							Some<span> content</span> rock &amp; roll
						</div>
						<buttton>Search</button>
					</body>
				</html>
				`
	assertExtractedText(c, content, "test title", `Some content rock & roll Search`)
}

func assertExtractedText(c *check.C, content, expectedTitle, expectedText string) {
	p := new(crawlerPayload)

	_, err := p.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	result, err := newTextExtractor().Process(context.TODO(), p)
	c.Assert(err, check.IsNil)

	c.Assert(result, check.DeepEquals, p)
	c.Assert(p.Title, check.Equals, expectedTitle)
	c.Assert(p.TextContent, check.Equals, expectedText)
}
