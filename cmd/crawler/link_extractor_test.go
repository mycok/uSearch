package crawler

import (
	"context"
	"net/url"
	"sort"

	"github.com/mycok/uSearch/cmd/crawler/mocks"

	"github.com/golang/mock/gomock"
	check "gopkg.in/check.v1"
)

// Initialize and register an instances of test suites to be
// executed by check testing package.
var (
	_ = check.Suite(new(LinkExtractorTestSuite))
	_ = check.Suite(new(ResolverURLTestSuite))
)

type ResolverURLTestSuite struct{}

func (s *ResolverURLTestSuite) TestResolveAbsoluteURL(c *check.C) {
	assertResolvedURL(
		c,
		"http://example.com/foo/",
		"/bar/baz",
		"http://example.com/bar/baz",
	)
}

func (s *ResolverURLTestSuite) TestResolveRelativeURL(c *check.C) {
	assertResolvedURL(
		c,
		"http://example.com/foo/",
		"bar/baz",
		"http://example.com/foo/bar/baz",
	)

	assertResolvedURL(
		c,
		"http://example.com/foo/",
		"/bar/baz",
		"http://example.com/bar/baz",
	)

	assertResolvedURL(
		c,
		"http://example.com/foo/secret/",
		"./bar/baz",
		"http://example.com/foo/secret/bar/baz",
	)

	assertResolvedURL(
		c,
		// Lack of a trailing slash means we should treat "secret" as a
		// file and the path is relative to its parent path.
		"http://example.com/foo/secret",
		"./bar/baz",
		"http://example.com/foo/bar/baz",
	)

	assertResolvedURL(
		c,
		"http://example.com/foo/secret/",
		// Every path denoted as ../ ignores the right most part of the base URL.
		"../../bar/baz",
		"http://example.com/bar/baz",
	)

	assertResolvedURL(
		c,
		"http://example.com/foo/secret/",
		// Every path denoted as ../ ignores the right most part of the base URL.
		"../bar/baz",
		"http://example.com/foo/bar/baz",
	)
}

func (s *ResolverURLTestSuite) TestResolveDoubleSlashURL(c *check.C) {
	assertResolvedURL(
		c,
		"https://www.myshop.com/users/",
		"//www.auth.com/users",
		"https://www.auth.com/users",
	)

	assertResolvedURL(
		c,
		"http://www.myshop.com/users/",
		"//www.auth.com/users",
		"http://www.auth.com/users",
	)

}

func assertResolvedURL(c *check.C, base, target, expected string) {
	relativeBase, err := url.Parse(base)
	c.Assert(err, check.IsNil)

	var resolvedURL string
	if resolved := resolveToAbsoluteURL(relativeBase, target); resolved != nil {
		resolvedURL = resolved.String()
	}
	c.Assert(resolvedURL, check.Equals, expected)
}

type LinkExtractorTestSuite struct {
	privateNetDetector *mocks.MockPrivateNetworkDetector
}

func (s *LinkExtractorTestSuite) TestLinkExtractor(c *check.C) {
	ctrl := gomock.NewController(c)
	s.privateNetDetector = mocks.NewMockPrivateNetworkDetector(ctrl)

	exp := s.privateNetDetector.EXPECT()
	exp.IsPrivate("example.com").Return(false, nil).Times(2)
	exp.IsPrivate("foo.com").Return(false, nil).Times(2)
	exp.IsPrivate("test.com").Return(false, nil).Times(3)

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
	s.assertExtractedLinks(c, "http://test.com", content, []string{
		"https://example.com",
		"http://foo.com",
		"http://test.com/absolute/link",
	}, []string{
		"http://test.com/local",
	})
}
func (s *LinkExtractorTestSuite) TestLinkExtractorWithNonHTTPLinks(c *check.C) {
	ctrl := gomock.NewController(c)
	s.privateNetDetector = mocks.NewMockPrivateNetworkDetector(ctrl)

	content := `
		<html>
		<body>
		<a href="ftp://example.com">An FTP site</a>
		</body>
		</html>
	`
	s.assertExtractedLinks(c, "http://test.com", content, nil, nil)
}

func (s *LinkExtractorTestSuite) TestLinkExtractorWithRelativeLinksToFile(c *check.C) {
	ctrl := gomock.NewController(c)
	s.privateNetDetector = mocks.NewMockPrivateNetworkDetector(ctrl)

	s.privateNetDetector.EXPECT().IsPrivate("test.com").Return(false, nil).Times(2)

	content := `
		<html>
		<body>
		<a href="./foo.html">link to foo</a>
		<a href="../private/data.html">login required</a>
		</body>
		</html>
		`
	// Note: Links to a file usually don't contain a trailing slash at the end of the url.
	s.assertExtractedLinks(c, "https://test.com/content/intro.html", content, []string{
		"https://test.com/content/foo.html",
		"https://test.com/private/data.html",
	}, nil)
}

func (s *LinkExtractorTestSuite) TestLinkExtractorWithRelativeLinksToDir(c *check.C) {
	ctrl := gomock.NewController(c)
	s.privateNetDetector = mocks.NewMockPrivateNetworkDetector(ctrl)

	s.privateNetDetector.EXPECT().IsPrivate("test.com").Return(false, nil).Times(2)

	content := `
		<html>
		<body>
		<a href="./foo.html">link to foo</a>
		<a href="../private/data.html">login required</a>
		</body>
		</html>
		`
	// Note: Links to a dir usually contain a trailing slash at the end of the url.
	s.assertExtractedLinks(c, "https://test.com/content/", content, []string{
		"https://test.com/content/foo.html",
		"https://test.com/private/data.html",
	}, nil)
}

func (s *LinkExtractorTestSuite) TestLinkExtractorWithBaseTag(c *check.C) {
	ctrl := gomock.NewController(c)
	s.privateNetDetector = mocks.NewMockPrivateNetworkDetector(ctrl)

	s.privateNetDetector.EXPECT().IsPrivate("test.com").Return(false, nil).Times(2)

	content := `
			<html>
				<head>
					<base href="https://test.com/base/"/>
				</head>
				<body>
					<a href="./foo.html">link to foo</a>
					<a href="../private/data.html">login required</a>
				</body>
			</html>
			`
	s.assertExtractedLinks(c, "https://testwithbaselink.com/content/", content, []string{
		"https://test.com/base/foo.html",
		"https://test.com/private/data.html",
	}, nil)
}

func (s *LinkExtractorTestSuite) TestLinkExtractorWithPrivateNetworkLinks(c *check.C) {
	ctrl := gomock.NewController(c)
	s.privateNetDetector = mocks.NewMockPrivateNetworkDetector(ctrl)

	exp := s.privateNetDetector.EXPECT()
	exp.IsPrivate("example.com").Return(false, nil)
	exp.IsPrivate("169.254.169.254").Return(true, nil)

	content := `
			<html>
				<body>
					<a href="https://example.com">link to foo</a>
					<a href="http://169.254.169.254/api/credentials"/>
				</body>
			</html>
	`
	s.assertExtractedLinks(c, "https://test.com/content/", content, []string{
		"https://example.com",
	}, nil)
}

func (s *LinkExtractorTestSuite) assertExtractedLinks(c *check.C, url, content string, expectedLinks []string, expectedNoFollowLinks []string) {
	p := &crawlerPayload{URL: url}
	_, err := p.RawContent.WriteString(content)
	c.Assert(err, check.IsNil)

	le := newLinkExtractor(s.privateNetDetector)

	result, err := le.Process(context.TODO(), p)
	c.Assert(err, check.IsNil)
	c.Assert(result, check.DeepEquals, p)

	sort.Strings(expectedLinks)
	sort.Strings(p.Links)

	c.Assert(p.Links, check.DeepEquals, expectedLinks)
	c.Assert(p.NoFollowLinks, check.DeepEquals, expectedNoFollowLinks)
}
