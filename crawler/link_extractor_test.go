package crawler

import (
	"context"
	"net/url"
	"sort"

	"github.com/golang/mock/gomock"
	check "gopkg.in/check.v1"

	mock_crawler "github.com/mycok/uSearch/crawler/mocks"
)

// Initialize and register pointer instances of test suites to be
// executed by check testing package.
var (
	_ = check.Suite(new(linkExtractionTestSuite))
	_ = check.Suite(new(resolveURLTestSuite))
)

type resolveURLTestSuite struct{}

func (s *resolveURLTestSuite) TestResolveLinkReferenceURL(c *check.C) {
	assertOnResolvedURL(
		c,
		"https://www.example.com/users",
		"//www.myshop.com/users",
		"https://www.myshop.com/users",
	)

	assertOnResolvedURL(
		c,
		"http://www.example.com/users",
		"//www.myshop.com/users",
		"http://www.myshop.com/users",
	)
}

func (s *resolveURLTestSuite) TestResolveAbsoluteURL(c *check.C) {
	assertOnResolvedURL(
		c,
		"https://www.example.com/users",
		"https://www.myshop.com/users",
		"https://www.myshop.com/users",
	)
}

func (s *resolveURLTestSuite) TestResolveRelativeURL(c *check.C) {
	assertOnResolvedURL(
		c,
		"http://example.com/foo/",
		"bar/baz",
		"http://example.com/foo/bar/baz",
	)

	assertOnResolvedURL(
		c,
		"http://example.com/foo/",
		"/bar/baz",
		"http://example.com/bar/baz",
	)

	assertOnResolvedURL(
		c,
		"http://example.com/foo/secret/",
		"./bar/baz",
		"http://example.com/foo/secret/bar/baz",
	)

	assertOnResolvedURL(
		c,
		// Lack of a trailing slash means we should treat "secret" as a
		// file and the path is relative to its parent path.
		"http://example.com/foo/secret",
		"./bar/baz",
		"http://example.com/foo/bar/baz",
	)
}

func assertOnResolvedURL(c *check.C, base, target, expected string) {
	baseURL, err := url.Parse(base)
	c.Assert(err, check.IsNil)

	var resolvedURL string
	if resolved := resolveToAbsoluteURL(baseURL, target); resolved != nil {
		resolvedURL = resolved.String()
	}

	c.Assert(resolvedURL, check.DeepEquals, expected)
}

type linkExtractionTestSuite struct {
	netDetector *mock_crawler.MockPrivateNetworkDetector
}

func (s *linkExtractionTestSuite) TestLinkExtractionWithNonHTTPLinks(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	s.netDetector = mock_crawler.NewMockPrivateNetworkDetector(ctrl)

	content := `
<html>
<body>
	<a href="ftp://example.com">An FTP site</a>
</body>
</html>
`
	payload := &crawlerPayload{URL: "http://test.com"}
	_, err := payload.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	s.assertOnExtractedLinks(c, payload, nil, nil)
}

func (s *linkExtractionTestSuite) TestLinkExtractionWithRelativeLinksToFile(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	s.netDetector = mock_crawler.NewMockPrivateNetworkDetector(ctrl)

	content := `
<html>
<body>
	<a href="./foo.html">link to foo</a>
	<a href="../private/data.html">login required</a>
</body>
</html>
`
	payload := &crawlerPayload{URL: "http://test.com/content/intro.html"}
	_, err := payload.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	s.assertOnExtractedLinks(c, payload, []string{
		"http://test.com/content/foo.html",
		"http://test.com/private/data.html",
	}, nil)
}

func (s *linkExtractionTestSuite) TestLinkExtractionWithRelativeLinksToDir(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	s.netDetector = mock_crawler.NewMockPrivateNetworkDetector(ctrl)

	content := `
<html>
<body>
	<a href="./foo.html">link to foo</a>
	<a href="../private/data.html">login required</a>
</body>
</html>
`
	payload := &crawlerPayload{URL: "http://test.com/content/"}
	_, err := payload.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	s.assertOnExtractedLinks(c, payload, []string{
		"http://test.com/content/foo.html",
		"http://test.com/private/data.html",
	}, nil)
}

func (s *linkExtractionTestSuite) TestLinkExtractionWithBaseTag(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	s.netDetector = mock_crawler.NewMockPrivateNetworkDetector(ctrl)

	content := `
<html>
<head><base href="https://test.com/base/"/>"/></head>
<body>
	<a href="./foo.html">link to foo</a>
	<a href="../private/data.html">login required</a>
</body>
</html>
`
	payload := &crawlerPayload{URL: "http://test.com/content/"}
	_, err := payload.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	s.assertOnExtractedLinks(c, payload, []string{
		"https://test.com/base/foo.html",
		"https://test.com/private/data.html",
	}, nil)
}

func (s *linkExtractionTestSuite) TestLinkExtractionWithPrivateNetworkLinks(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	s.netDetector = mock_crawler.NewMockPrivateNetworkDetector(ctrl)

	exp := s.netDetector.EXPECT()
	exp.IsNetworkPrivate("example.com").Return(false, nil)
	exp.IsNetworkPrivate("169.254.169.254").Return(true, nil)

	content := `
<html>
<body>
	<a href="https://example.com">link to foo</a>
	<a href="http://169.254.169.254/api/credentials">login required</a>
</body>
</html>
`

	payload := &crawlerPayload{URL: "http://test.com/content/"}
	_, err := payload.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	s.assertOnExtractedLinks(c, payload, []string{
		"https://example.com",
	}, nil)
}

func (s *linkExtractionTestSuite) TestLinkExtractionSuccessfulRun(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	s.netDetector = mock_crawler.NewMockPrivateNetworkDetector(ctrl)

	exp := s.netDetector.EXPECT()
	exp.IsNetworkPrivate("example.com").Return(false, nil).Times(2)
	exp.IsNetworkPrivate("foo.com").Return(false, nil).Times(2)

	content := `
<html>
	<body>
		<a href="https://example.com"/>
		<a href="//foo.com"></a>
		<a href="/absolute/link"></a>

		<!-- the following link should be included in the no follow link list -->
		<a href="./local" rel="nofollow"></a>

		<!-- duplicates, even with fragments should be skipped -->
		<a href="https://example.com#important"/>
		<a href="//foo.com"></a>
		<a href="/absolute/link#some-anchor"></a>
	</body>
</html>
`
	payload := &crawlerPayload{URL: "http://test.com"}
	_, err := payload.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	expectedLinks := []string{
		"https://example.com",
		"http://foo.com",
		"http://test.com/absolute/link",
	}

	expectedNoFollowLinks := []string{
		"http://test.com/local",
	}

	s.assertOnExtractedLinks(c, payload, expectedLinks, expectedNoFollowLinks)
}

func (s *linkExtractionTestSuite) assertOnExtractedLinks(
	c *check.C, payload *crawlerPayload,
	expectedLinks []string, expectedNoFollowLinks []string,
) {

	le := newLinkExtractor(s.netDetector)
	processedPayload, err := le.Process(context.TODO(), payload)
	c.Assert(err, check.IsNil)
	c.Assert(processedPayload, check.DeepEquals, payload)

	sort.Strings(payload.Links)
	sort.Strings(expectedLinks)
	c.Assert(payload.Links, check.DeepEquals, expectedLinks)
	c.Assert(payload.NoFollowLinks, check.DeepEquals, expectedNoFollowLinks)
}
