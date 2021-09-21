package crawler

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mycok/uSearch/cmd/crawler/mocks"

	"github.com/golang/mock/gomock"
	check "gopkg.in/check.v1"
)

// Initialize and register an instance of the LinkFetcherTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(LinkFetcherTestSuite))

type LinkFetcherTestSuite struct {
	urlGetter          *mocks.MockURLGetter
	privateNetDetector *mocks.MockPrivateNetworkDetector
}

func (s *LinkFetcherTestSuite) SetUpTest(c *check.C) {
	ctrl := gomock.NewController(c)
	s.urlGetter = mocks.NewMockURLGetter(ctrl)
	s.privateNetDetector = mocks.NewMockPrivateNetworkDetector(ctrl)
}

func (s *LinkFetcherTestSuite) TestLinkFetcherWithXcludedXtension(c *check.C) {
	p := s.fetchLink(c, "http://example.com/bar.png")
	c.Assert(p, check.IsNil)
}

func (s *LinkFetcherTestSuite) TestLinkFetcher(c *check.C) {
	s.privateNetDetector.EXPECT().IsPrivate("example.com").Return(false, nil)
	s.urlGetter.EXPECT().Get("http://example.com/index.html").Return(
		makeResponse(200, "yea!, test passed", "application/xhtml"),
		nil,
	)

	p := s.fetchLink(c, "http://example.com/index.html")
	c.Assert(p.RawContent.String(), check.Equals, "yea!, test passed")
}

func (s *LinkFetcherTestSuite) TestLinkFetcherForLinkWithPortNumber(c *check.C) {
	s.privateNetDetector.EXPECT().IsPrivate("example.com").Return(false, nil)
	s.urlGetter.EXPECT().Get("http://example.com:3456").Return(
		makeResponse(200, "hello", "application/xhtml"),
		nil,
	)

	p := s.fetchLink(c, "http://example.com/3456")
	c.Assert(p.RawContent.String(), check.Equals, "hello")
}

func (s *LinkFetcherTestSuite) TestLinkFetcherWithWrongStatusCode(c *check.C) {
	s.privateNetDetector.EXPECT().IsPrivate("example.com").Return(false, nil)
	s.urlGetter.EXPECT().Get("http://example.com/index.html").Return(
		makeResponse(400, `{"error": "something went wrong"}`, "application/json"),
		nil,
	)

	p := s.fetchLink(c, "http://example.com/index.html")
	c.Assert(p, check.IsNil)
}

func (s *LinkFetcherTestSuite) TestLinkFetcherWithNonHTMLContentType(c *check.C) {
	s.privateNetDetector.EXPECT().IsPrivate("example.com").Return(false, nil)
	s.urlGetter.EXPECT().Get("http://example.com/users").Return(
		makeResponse(200, `{"users": "[]"}`, "application/json"),
		nil,
	)

	p := s.fetchLink(c, "http://example.com/users")
	c.Assert(p, check.IsNil)
}

func (s *LinkFetcherTestSuite) TestLinkFetcherWithUrlFromAPrivateNet(c *check.C) {
	s.privateNetDetector.EXPECT().IsPrivate("169.254.169.251").Return(true, nil)

	p := s.fetchLink(c, "http:169.254.169.251/login")
	c.Assert(p, check.IsNil)
}

// Test helpers
func (s *LinkFetcherTestSuite) fetchLink(c *check.C, urlStr string) *crawlerPayload {
	p := &crawlerPayload{URL: urlStr}
	output, err := newLinkFetcher(s.urlGetter, s.privateNetDetector).Process(context.TODO(), p)
	c.Assert(err, check.IsNil)

	if output != nil {
		c.Assert(output, check.FitsTypeOf, p)

		return output.(*crawlerPayload)
	}

	return nil
}

func makeResponse(status int, body, contentType string) *http.Response {
	res := new(http.Response)
	res.Body = ioutil.NopCloser(strings.NewReader(body))
	res.StatusCode = status
	if contentType != "" {
		res.Header = make(http.Header)
		res.Header.Set("Content-Type", contentType)
	}

	return res
}
