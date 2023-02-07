package crawler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"time"

	"github.com/google/uuid"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/crawler"
	"github.com/mycok/uSearch/crawler/privnet"
	"github.com/mycok/uSearch/linkgraph/graph"
	memGraphStore "github.com/mycok/uSearch/linkgraph/store/memory"
	memTxtIndex "github.com/mycok/uSearch/textindexer/store/memory"
)

// Initialize and register a pointer instance of the crawlerIntegrationTestSuite
// to be executed by check testing package.
var _ = check.Suite(new(crawlerIntegrationTestSuite))

var serverRes = `
<html>
<head>
	<title>A title</title>
	<base href="http://google.com/"/>
</head>
<body>
  <a href="./relative">I am a link relative to base</a>
  <a href="/absolute/path">I am an absolute link</a>
  <a href="//images/cart.png">I am using the same URL scheme as this page</a>
  
  <!-- Link should be added to the index but without creating an edge to it -->
  <a href="ignore-me" rel="nofollow"/>
  <!-- The following links should be ignored -->
  <a href="file:///etc/passwd"></a>
  <a href="http://169.254.169.254/api/credentials">Link-local address</a>
</body>
</html>
`

type crawlerIntegrationTestSuite struct{}

func (s *crawlerIntegrationTestSuite) TestCrawlerPipeline(c *check.C) {
	linkGraph := memGraphStore.NewInMemoryGraph()

	idx, err := memTxtIndex.NewInMemoryIndex()
	if err != nil {
		c.Fatal("Failed to initialize memory index: ", err)
	}

	netDetector, err := privnet.NewDetectorFromCIDRs("169.254.0.0/16")
	if err != nil {
		c.Fatal("Failed to initialize private network detector: ", err)
	}

	// Create a configuration object for the crawler
	cfg := crawler.Config{
		PrivateNetworkDetector: netDetector,
		URLGetter:              http.DefaultClient,
		Graph:                  linkGraph,
		Indexer:                idx,
		NumOfFetchWorkers:      5,
	}

	// Start a TLS server and a regular server
	srv1 := createAndAssertOnTestServer(c)
	srv2 := createAndAssertOnTestServer(c)
	defer srv1.Close()
	defer srv2.Close()

	// Upsert links into the linkgraph store.
	urls := []string{srv1.URL, srv2.URL}
	for _, url := range urls {
		c.Logf("upserting %q into the graph store", url)

		err := linkGraph.UpsertLink(&graph.Link{URL: url})
		c.Assert(err, check.IsNil, check.Commentf("upserting %q failed", url))
	}

	// Retrieve a link iterator to serve as the source object of
	// the pipeline.
	linkIt := createAndAssertOnIterator(c, linkGraph)

	// Create, execute and assert on the crawler.
	count, err := crawler.New(cfg).Crawl(context.TODO(), linkIt)
	c.Assert(err, check.IsNil)
	c.Assert(count, check.Equals, 2)

	// Assert that upsert links after the entire crawl process match the expected
	// crawled data.
	expected := []string{
		srv1.URL,
		srv2.URL,
		"http://google.com/absolute/path",
		"http://google.com/relative",
		"http://google.com/ignore-me",
	}
	var obtained []string
	var forIndexing []uuid.UUID

	// Retrieve a new instance of a link iterator.
	linkIt = createAndAssertOnIterator(c, linkGraph)
	for linkIt.Next() {
		obtained = append(obtained, linkIt.Link().URL)
		if linkIt.Link().URL == srv1.URL || linkIt.Link().URL == srv2.URL {
			forIndexing = append(forIndexing, linkIt.Link().ID)
		}
	}
	sort.Strings(obtained)
	sort.Strings(expected)
	c.Assert(obtained, check.DeepEquals, expected)

	// Assert that all links from the link graph store are indexed.
	zeroTime := time.Time{}
	expectedTitle := "A title"
	expectedContent := "I am a link relative to base I am an absolute link I am using the same URL scheme as this page Link-local address"

	for _, id := range forIndexing {
		doc, err := idx.FindByID(id)

		c.Assert(err, check.IsNil)
		c.Assert(doc.Title, check.Equals, expectedTitle)
		c.Assert(doc.Content, check.Equals, expectedContent)
		c.Assert(doc.IndexedAt.After(zeroTime), check.Equals, true)
	}
}

func createAndAssertOnTestServer(c *check.C) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Logf("GET %q", r.URL)
		w.Header().Set("Content-Type", "application/xhtml")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte(serverRes))
		c.Assert(err, check.IsNil)
	}))

	return srv
}

func createAndAssertOnIterator(c *check.C, graph graph.Graph) graph.LinkIterator {
	linkIt, err := graph.Links(
		uuid.Nil, uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		time.Now(),
	)
	if err != nil {
		c.Fatal("Failed to retrieve link iterator: ", err)
	}

	return linkIt
}
