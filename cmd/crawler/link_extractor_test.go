package crawler

import (
	// "context"
	"net/url"
	// "sort"

	// "github.com/mycok/uSearch/cmd/crawler/mocks"

	// "github.com/golang/mock/gomock"
	check "gopkg.in/check.v1"
)

// Initialize and register an instances of test suites to be
// executed by check testing package.
var (
	_ = check.Suite(new(LinkExtractorSuite))
	_ = check.Suite(new(ResolverURLTestSuite))
)

type ResolverURLTestSuite struct {}

type LinkExtractorSuite struct {

}

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