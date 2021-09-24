package crawler

import (
	"context"

	"github.com/mycok/uSearch/internal/graphlink/graph"
	"github.com/mycok/uSearch/internal/pipeline"
)

// Crawler implements a web-page crawling pipeline consisting of the following
// stages:
//
// - Given a URL, retrieve the web-page contents from the remote server.
// - Extract and resolve absolute and relative links from the retrieved page.
// - Extract page title and text content from the retrieved page.
// - Update the link graph: add new links and create edges between the crawled
//   page and the links within it.
// - Index crawled page title and text content.
type Crawler struct {
	p *pipeline.Pipeline
}

// NewCrawler returns a new crawler instance.
func NewCrawler(cfg Config) *Crawler {
	return &Crawler{p: assembleCrawlerPipeline(cfg)}
}

func assembleCrawlerPipeline(cfg Config) *pipeline.Pipeline {
	return pipeline.New(
		pipeline.FixedWorkerPool(
			newLinkFetcher(cfg.URLGetter, cfg.PrivateNetworkDetector),
			cfg.NumOfFetchWorkers,
		),
		pipeline.FIFO(newLinkExtractor(cfg.PrivateNetworkDetector)),
		pipeline.FIFO(newTextExtractor()),
		pipeline.Broadcast(
			newGraphUpdater(cfg.Graph),
			newTextIndexer(cfg.Indexer),
		),
	)
}

// Crawl iterates linkIterator and sends each link through the crawler pipeline
// returning the total count of links that went through the pipeline. Calls to
// Crawl block until the link iterator is exhausted, an error occurs or the
// context is cancelled.
func (c *Crawler) Crawl(ctx context.Context, linkIt graph.LinkIterator) (int, error) {
	sink := new(sink)
	err := c.p.Process(ctx, &linkSource{linkIt: linkIt}, sink)

	return sink.getCount(), err
}
