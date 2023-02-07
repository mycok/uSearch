package crawler

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/golang/mock/gomock"
	check "gopkg.in/check.v1"

	mock_crawler "github.com/mycok/uSearch/crawler/mocks"
)

// Initialize and register a pointer instance of the linkFetchingTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(linkFetchingTestSuite))

type linkFetchingTestSuite struct {
	urlGetter   *mock_crawler.MockURLGetter
	netDetector *mock_crawler.MockPrivateNetworkDetector
}

func (s *linkFetchingTestSuite) SetUpTest(c *check.C) {
	ctrl := gomock.NewController(c)

	s.urlGetter = mock_crawler.NewMockURLGetter(ctrl)
	s.netDetector = mock_crawler.NewMockPrivateNetworkDetector(ctrl)
}

func (s *linkFetchingTestSuite) TearDownTest(c *check.C) {
	s.urlGetter = nil
	s.netDetector = nil
}

func (s *linkFetchingTestSuite) TestLinkFetchingWithInvalidURLXtention(c *check.C) {
	payload := s.fetchLink(c, "http://example.com/bar.jpg")
	c.Assert(payload, check.IsNil)
}

func (s *linkFetchingTestSuite) TestLinkFetchingWithPrivateNetworkURL(c *check.C) {
	s.netDetector.EXPECT().IsNetworkPrivate("169.254.169.254").Return(true, nil)

	payload := s.fetchLink(c, "169.254.169.254")
	c.Assert(payload, check.IsNil)
}

func (s *linkFetchingTestSuite) TestLinkFetchingForURLWithPortNumber(c *check.C) {
	s.netDetector.EXPECT().IsNetworkPrivate("example.com").Return(false, nil)
	s.urlGetter.EXPECT().Get("http://example.com:1234/index.html").Return(makeResponse(
		200, "application/html", "hello",
	), nil)

	payload := s.fetchLink(c, "http://example.com:1234/index.html")
	c.Assert(payload.RawContent.String(), check.DeepEquals, "hello")
}

func (s *linkFetchingTestSuite) TestLinkFetchingWithInvalidStatusCode(c *check.C) {
	s.netDetector.EXPECT().IsNetworkPrivate("example.com").Return(false, nil)
	s.urlGetter.EXPECT().Get("example.com/index.html").Return(makeResponse(
		500, "application/json", `{"error": "internal server error"}`,
	), nil)

	payload := s.fetchLink(c, "example.com/index.html")
	c.Assert(payload, check.IsNil)
}

func (s *linkFetchingTestSuite) TestLinkFetchingWithNonHTMLContentType(c *check.C) {
	s.netDetector.EXPECT().IsNetworkPrivate("example.com").Return(false, nil)
	s.urlGetter.EXPECT().Get("http://example.com/list/products").Return(makeResponse(
		200, "application/json", `{"products": "[a, b, c]"}`,
	))

	payload := s.fetchLink(c, "http://example.com/list/products")
	c.Assert(payload, check.IsNil)
}

func (s *linkFetchingTestSuite) TestLinkFetchingSuccessfulRun(c *check.C) {
	s.netDetector.EXPECT().IsNetworkPrivate("example.com").Return(false, nil)
	s.urlGetter.EXPECT().Get("http://example.com/index.html").Return(makeResponse(
		200, "application/html", "yea!!, test passed",
	), nil)

	payload := s.fetchLink(c, "http://example.com/index.html")
	c.Assert(payload.RawContent.String(), check.DeepEquals, "yea!!, test passed")
}

func (s *linkFetchingTestSuite) fetchLink(c *check.C, url string) *crawlerPayload {
	payload := &crawlerPayload{URL: url}
	f := newLinkFetcher(s.urlGetter, s.netDetector)
	processedPayload, err := f.Process(context.TODO(), payload)
	c.Assert(err, check.IsNil)

	if processedPayload != nil {
		c.Assert(processedPayload, check.FitsTypeOf, payload)

		return processedPayload.(*crawlerPayload)
	}

	return nil
}

func makeResponse(code int, contentType, body string) *http.Response {
	resp := new(http.Response)
	resp.Body = io.NopCloser(strings.NewReader(body))
	resp.StatusCode = code

	if contentType != "" {
		resp.Header = make(http.Header)
		resp.Header.Set("Content-Type", contentType)
	}

	return resp
}
