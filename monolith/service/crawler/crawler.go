package crawler

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	crawler_pipeline "github.com/mycok/uSearch/crawler"
	"github.com/mycok/uSearch/monolith/partition"
)

// Service represents a web-crawler service for the uSearch application. it
// satisfies the service.Service interface.
type Service struct {
	config  Config
	crawler *crawler_pipeline.Crawler
}

// New creates and returns a fully configured web-crawler service instance.
func New(config Config) (*Service, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("crawler service: config validation failed: %w", err)
	}

	return &Service{
		config: config,
		crawler: crawler_pipeline.New(crawler_pipeline.Config{
			PrivateNetworkDetector: config.PrivateNetworkDetector,
			URLGetter:              config.URLGetter,
			Graph:                  config.GraphAPI,
			Indexer:                config.IndexAPI,
			NumOfFetchWorkers:      config.NumOfFetchWorkers,
		}),
	}, nil
}

// Name returns the name of the service.
func (svc *Service) Name() string { return "crawler" }

// Run executes the service and blocks until the context gets cancelled
// or an error occurs.
func (svc *Service) Run(ctx context.Context) error {
	svc.config.Logger.WithField(
		"update_interval", svc.config.CrawlUpdateInterval.String(),
	).Info("starting service")
	defer svc.config.Logger.Info("stopped service")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-svc.config.Clock.After(svc.config.CrawlUpdateInterval):
			// After the crawl update interval elapses, retrieve the current
			// partition number / position for the service for use in the next
			// crawl pass.
			currPartition, numOfPartitions, err := svc.config.PartitionDetector.PartitionInfo()
			if err != nil {
				if errors.Is(err, partition.ErrNoPartitionDataAvailableYet) {
					svc.config.Logger.Warn(
						"deferring crawler update pass: partition data not yet available",
					)

					continue
				}

				return err
			}

			if err := svc.crawlGraph(ctx, currPartition, numOfPartitions); err != nil {
				return err
			}
		}
	}
}

func (svc *Service) crawlGraph(
	ctx context.Context, currPartition, numOfPartitions int,
) error {

	svc.config.Logger.Info("started crawler update pass")

	fullRange, err := partition.NewFullRange(numOfPartitions)
	if err != nil {
		return fmt.Errorf("crawler: unable to compute ID ranges for partition: %w", err)
	}

	fromID, toID, err := fullRange.PartitionRange(currPartition)
	if err != nil {
		return fmt.Errorf("crawler: unable to compute ID ranges for partition: %w", err)
	}

	svc.config.Logger.WithFields(logrus.Fields{
		"partition":         currPartition,
		"num_of_partitions": numOfPartitions,
	}).Info("starting new crawl pass")

	startedAt := svc.config.Clock.Now()

	// Retrieve the links iterator for use as the crawler link source.
	linkIt, err := svc.config.GraphAPI.Links(
		fromID, toID, svc.config.Clock.Now().Add(-svc.config.ReIndexThreshold),
	)
	if err != nil {
		return fmt.Errorf("crawler: unable to retrieve links iterator: %w", err)
	}

	numOfProccessed, err := svc.crawler.Crawl(ctx, linkIt)
	if err != nil {
		return fmt.Errorf("crawler: unable to complete crawling the link graph: %w", err)
	}

	if err = linkIt.Close(); err != nil {
		return fmt.Errorf("crawler: unable to complete crawling the link graph: %w", err)
	}

	svc.config.Logger.WithFields(logrus.Fields{
		"processed_link_count": numOfProccessed,
		"elapsed_time":         svc.config.Clock.Now().Sub(startedAt).String(),
	}).Info("successfully completed crawl pass")

	return nil
}
