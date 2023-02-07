package crawler

import (
	"context"

	check "gopkg.in/check.v1"
)

// Initialize and register a pointer instance of the textExtractionTestSuite
// to be executed by check testing package.
var _ = check.Suite(new(textExtractionTestSuite))

type textExtractionTestSuite struct{}

func (s *textExtractionTestSuite) TestTextExtractionSuccessfulRun(c *check.C) {
	content := `<div>Some<span> content</span> rock &amp; roll</div>
<buttton>Search</button>
`
	payload := &crawlerPayload{}
	_, err := payload.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	assertOnExtractedText(c, payload, "", "Some content rock & roll Search")
}

func (s *textExtractionTestSuite) TestContentAndTitleExtraction(c *check.C) {
	content := `<html>
<head>
<title>Test title</title>
</head>
<body>
<div>Some<span> content</span></div>
</body>
</html>
`
	payload := &crawlerPayload{}
	_, err := payload.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	assertOnExtractedText(c, payload, "Test title", "Some content")
}

func assertOnExtractedText(
	c *check.C, payload *crawlerPayload,
	expectedTitle, expectedContent string,
) {

	te := newTextExtractor()
	processedPayload, err := te.Process(context.TODO(), payload)
	c.Assert(err, check.IsNil)
	c.Assert(processedPayload, check.DeepEquals, payload)
	c.Assert(payload.Title, check.Equals, expectedTitle)
	c.Assert(payload.TextContent, check.Equals, expectedContent)
}
