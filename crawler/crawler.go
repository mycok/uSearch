/*
	crawler package is a web crawler implementation using a pipeline strategy
	to crawl, extract, store and index web content. The crawling process involve
	the following stages:
		1. Given a URL, retrieve the web-page contents from the remote server.
		2. Extract and resolve all links from the retrieved page content.
		3. Extract page title and text content from the retrieved page raw HTML
		content.
		4. Update the link graph store by:
			- updating the original / source link with new data.
			- creating and adding new link objects from the newly
			  discovered links.
			- creating and adding new edge objects between the original / source
			  link and newly discovered links.
			- index the original / source link page text content.
*/

package crawler

import (
	"context"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/pipeline"
)

// Config serves as a configuration object for the crawler.
type Config struct {
	PrivateNetworkDetector PrivateNetworkDetector
	URLGetter              URLGetter
	Graph                  MiniGraph
	Indexer                MiniIndexer
	NumOfFetchWorkers      int
}

// Crawler executes a web crawler pipeline.
type Crawler struct {
	p *pipeline.Pipeline
}

// New configures and returns pointer to a fully configured crawler type.
func New(cfg Config) *Crawler {
	return &Crawler{p: assembleCrawlerPipeline(cfg)}
}

func assembleCrawlerPipeline(cfg Config) *pipeline.Pipeline {
	return pipeline.New(
		pipeline.NewFixedWorkerPool(
			newLinkFetcher(cfg.URLGetter, cfg.PrivateNetworkDetector),
			cfg.NumOfFetchWorkers,
		),
		pipeline.NewFIFO(
			newLinkExtractor(cfg.PrivateNetworkDetector),
		),
		pipeline.NewFIFO(newTextExtractor()),
		pipeline.NewBroadcastWorkerPool(
			newGraphUpdater(cfg.Graph),
			newTextIndexer(cfg.Indexer),
		),
	)
}

// Crawl executes the pipeline. calls to crawl block until the pipeline
// execution is complete.
func (c *Crawler) Crawl(
	ctx context.Context, linkIt graph.LinkIterator,
) (int, error) {

	sink := new(countingSink)

	err := c.p.Execute(ctx, &linkSource{linkIt}, sink)

	return sink.getCount(), err
}
