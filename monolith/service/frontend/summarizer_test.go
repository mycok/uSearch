package frontend

import (
	"bufio"
	"strings"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(SummarizerTestSuite))

type SummarizerTestSuite struct{}

func (s *SummarizerTestSuite) TestScanSentence(c *check.C) {
	input := "Dot without space.Ending in ! Ending in $foo$.1 number at start or end 1. Question?"
	expected := []string{
		"Dot without space.",
		"Ending in !",
		" Ending in $foo$",
		"1 number at start or end 1.",
		" Question?",
	}

	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(scanSentence)

	got := []string{}
	for scanner.Scan() {
		got = append(got, scanner.Text())
	}

	c.Assert(expected, check.DeepEquals, got)
}

func (s *SummarizerTestSuite) TestSummary(c *check.C) {
	input := `
Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium
doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore
veritatis et quasi architecto beatae vitae dicta sunt explicabo. Nemo enim
ipsam voluptatem quia voluptas KEYWORD1 sit aspernatur aut odit aut fugit, sed quia
consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt. Neque
porro quisquam est, qui dolorem ipsum quia dolor sit amet, consectetur,
adipisci velit, sed quia non numquam eius modi tempora incidunt ut labore et
dolore magnam aliquam quaerat voluptatem. Ut enim ad KEYWORD2 minima veniam, quis
nostrum exercitationem ullam corporis suscipit laboriosam, nisi ut aliquid ex
ea commodi consequatur? Quis autem vel eum iure reprehenderit qui in ea
voluptate velit esse quam nihil molestiae consequatur, vel illum qui dolorem
eum fugiat quo voluptas nulla pariatur?`

	expectedSummary := `
Nemo enim
ipsam voluptatem quia voluptas KEYWORD1 sit aspernatur aut odit aut fugit, sed quia
consequuntur magni ..... Ut enim ad KEYWORD2 minima veniam, quis
nostrum exercitationem ullam corporis suscipit laboriosam, nisi ut aliquid ex
ea commodi consequatur?.`[1:]

	summarizer := newMatchSummarizer("KEYWORD1 KEYWORD2", 256)
	summary := summarizer.Summary(input)

	c.Assert(summary, check.Equals, expectedSummary)
}
