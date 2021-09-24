package crawler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"time"

	"github.com/mycok/uSearch/cmd/crawler"
	"github.com/mycok/uSearch/cmd/crawler/detector"
	"github.com/mycok/uSearch/internal/graphlink/graph"
	memGraph "github.com/mycok/uSearch/internal/graphlink/store/memory"
	"github.com/mycok/uSearch/internal/textindexer/index"
	memIndexer "github.com/mycok/uSearch/internal/textindexer/store/memory"

	"github.com/google/uuid"
	check "gopkg.in/check.v1"
)

var (
	_ = check.Suite(new(CrawlerIntegrationTestSuite))

	serverRes = `
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
)

type CrawlerIntegrationTestSuite struct{}

func (s *CrawlerIntegrationTestSuite) TestCrawlerPipeline(c *check.C) {
	graphLinkStore := memGraph.NewInMemoryGraph()
	txtIndexer := assertCreateBleveIndexer(c)

	cfg := crawler.Config{
		PrivateNetworkDetector: assertCreatePrivateNetworkDetector(c),
		Graph:                  graphLinkStore,
		Indexer:                txtIndexer,
		URLGetter:              http.DefaultClient,
		NumOfFetchWorkers:      5,
	}

	// Start a TLS server and a regular server
	srv1 := assertCreateTestServer(c)
	srv2 := assertCreateTestServer(c)
	defer srv2.Close()
	defer srv1.Close()

	assertUpsertLinks(c, graphLinkStore, []string{
		srv1.URL,
		srv2.URL,
	})

	count, err := crawler.NewCrawler(cfg).Crawl(context.Background(), assertReturnLinkAlterator(c, graphLinkStore))
	c.Assert(err, check.IsNil)
	c.Assert(count, check.Equals, 2)

	assertGraphLinksMatchList(c, graphLinkStore, []string{
		srv1.URL,
		srv2.URL,
		"http://google.com/absolute/path",
		"http://google.com/relative",
		"http://google.com/ignore-me",
	})

	assertLinksAreStoredAndIndexed(c, graphLinkStore, txtIndexer, []string{
		srv1.URL,
		srv2.URL,
	}, "A title",
		"I am a link relative to base I am an absolute link I am using the same URL scheme as this page Link-local address",
	)
}

func assertLinksAreStoredAndIndexed(c *check.C, g graph.Graph, i index.Indexer, urlLinks []string, expectedTitle, expectedContent string) {
	var urlToID = make(map[string]uuid.UUID)
	// Retrieve the link objects from the linkgraph store and store them in the [urlToID] map.
	for it := assertReturnLinkAlterator(c, g); it.Next(); {
		link := it.Link()
		urlToID[link.URL] = link.ID
	}

	zeroTime := time.Time{}

	for _, url := range urlLinks {
		// Check whether the provided urlLinks are all stored in the graphlink store by matching
		// them with the url's stored in the urlToID map.
		id, exists := urlToID[url]
		c.Assert(exists, check.Equals, true, check.Commentf("linkUrl %q was not retrieved", url))

		// Check the textindexer store for a parallel document matching the links from the linkgraph store.
		doc, err := i.FindByID(id)
		c.Assert(err, check.IsNil)

		c.Assert(doc.Title, check.Equals, expectedTitle)
		c.Assert(doc.Content, check.Equals, expectedContent)
		c.Assert(doc.IndexedAt.After(zeroTime), check.Equals, true, check.Commentf("indexed document with zero indexedAt timestamp"))

	}
}

func assertGraphLinksMatchList(c *check.C, g graph.Graph, expectedLinks []string) {
	var obtained []string
	for it := assertReturnLinkAlterator(c, g); it.Next(); {
		obtained = append(obtained, it.Link().URL)
	}

	sort.Strings(obtained)
	sort.Strings(expectedLinks)

	c.Assert(obtained, check.DeepEquals, expectedLinks)
}

func assertUpsertLinks(c *check.C, g graph.Graph, urlLinks []string) {
	for _, l := range urlLinks {
		err := g.UpsertLink(&graph.Link{URL: l})

		c.Logf("upserting %q into a graph", l)
		c.Assert(err, check.IsNil, check.Commentf("inserting %q", l))
	}
}

func assertCreateBleveIndexer(c *check.C) *memIndexer.InMemoryBleveIndexer {
	idxer, err := memIndexer.NewInMemoryBleveIndexer()
	c.Assert(err, check.IsNil)

	return idxer
}

func assertCreatePrivateNetworkDetector(c *check.C) *detector.Detector {
	det, err := detector.NewDetectorFromCIDRs("169.254.0.0/16")
	c.Assert(err, check.IsNil)

	return det
}

func assertReturnLinkAlterator(c *check.C, g graph.Graph) graph.LinkIterator {
	it, err := g.Links(uuid.Nil, uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"), time.Now())
	c.Assert(err, check.IsNil)

	return it
}

func assertCreateTestServer(c *check.C) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		c.Logf("GET %q", r.URL)
		rw.Header().Set("Content-Type", "application/xhtml")
		rw.WriteHeader(http.StatusOK)

		_, err := rw.Write([]byte(serverRes))
		c.Assert(err, check.IsNil)
	}))

	return srv
}
