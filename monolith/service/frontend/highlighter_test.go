package frontend

import (
	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(HighlighterTestSuite))

type HighlighterTestSuite struct{}

func (s *HighlighterTestSuite) TestSentenceHighlight(c *check.C) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "Test KEYWORD1",
			expected: "Test <em>KEYWORD1</em>",
		},
		{
			input:    "Data. KEYWORD2 lorem ipsum.KEYWORD1",
			expected: "Data. <em>KEYWORD2</em> lorem ipsum.<em>KEYWORD1</em>",
		},
		{
			input:    "no match",
			expected: "no match",
		},
	}

	h := newMatchHighlighter("KEYWORD1 KEYWORD2")

	for index, tc := range testCases {
		c.Logf("spec %d", index)
		got := h.Highlight(tc.input)
		c.Assert(tc.expected, check.Equals, got)
	}
}
