package pagerank

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/mycok/uSearch/bsp"
	"github.com/mycok/uSearch/monolith/partition"
	"github.com/mycok/uSearch/pagerank"
)

// Service represents a page-rank service for the uSearch application. it
// satisfies the service.Service interface.
type Service struct {
	config     Config
	calculator *pagerank.Calculator
}

// New creates and returns a fully configured page-rank service instance.
func New(config Config) (*Service, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("page-rank service: config validation failed: %w", err)
	}

	calc, err := pagerank.NewCalculator(pagerank.Config{
		ComputeWorkers: config.NumOfComputeWorkers,
	})
	if err != nil {
		return nil, fmt.Errorf("page-rank service: config validation failed: %w", err)
	}

	return &Service{
		config:     config,
		calculator: calc,
	}, nil
}

// Name returns the name of the service.
func (svc *Service) Name() string { return "page-rank" }

// Run executes the service and blocks until the context gets cancelled
// or an error occurs.
func (svc *Service) Run(ctx context.Context) error {
	svc.config.Logger.WithField(
		"update_interval", svc.config.UpdateInterval.String(),
	).Info("started service")
	defer svc.config.Logger.Info("stopped service")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-svc.config.Clock.After(svc.config.UpdateInterval):
			// After the update interval elapses, retrieve the current
			// partition number / position for the service for use in the next
			// page-rank pass.
			currPartition, _, err := svc.config.PartitionDetector.PartitionInfo()
			if err != nil {
				if errors.Is(err, partition.ErrNoPartitionDataAvailableYet) {
					svc.config.Logger.Warn(
						"deferring page-rank update pass: partition data not yet available",
					)

					continue
				}

				return err
			}

			if currPartition != 0 {
				svc.config.Logger.Info(
					"service should only run on the master node of the application cluster",
				)

				return nil
			}

			if err := svc.updateGraphScores(ctx); err != nil {
				return err
			}
		}
	}
}

func (svc *Service) updateGraphScores(ctx context.Context) error {
	svc.config.Logger.Info("started page-rank update pass")

	startedAt := svc.config.Clock.Now()
	maxUUID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	tick := svc.config.Clock.Now()
	if err := svc.calculator.Graph().Reset(); err != nil {
		return err
	}

	if err := svc.loadLinks(uuid.Nil, maxUUID, startedAt); err != nil {
		return err
	}

	if err := svc.loadEdges(uuid.Nil, maxUUID, startedAt); err != nil {
		return err
	}
	graphPopulationDuration := svc.config.Clock.Now().Sub(tick)

	tick = svc.config.Clock.Now()
	if err := svc.calculator.CalculatePageRanks(ctx); err != nil {
		return err
	}
	scoreCalculationDuration := svc.config.Clock.Now().Sub(tick)

	tick = svc.config.Clock.Now()
	if err := svc.calculator.Scores(svc.persistScores); err != nil {
		return err
	}
	scorePersistanceDuration := svc.config.Clock.Now().UTC().Sub(tick)

	svc.config.Logger.WithFields(logrus.Fields{
		"processed_links":            len(svc.calculator.Graph().Vertices()),
		"graph_population_duration":  graphPopulationDuration,
		"score_calculation_duration": scoreCalculationDuration,
		"score_persistance_duration": scorePersistanceDuration,
		"total_processing_time":      svc.config.Clock.Now().Sub(startedAt),
	})

	return nil
}

func (svc *Service) persistScores(vertexID string, score float64) error {
	linkID, err := uuid.Parse(vertexID)
	if err != nil {
		return err
	}

	return svc.config.IndexAPI.UpdateScore(linkID, score)
}

func (svc *Service) loadLinks(fromID, toID uuid.UUID, retrievedBefore time.Time) error {
	// Retrieve links iterator for the queried partition from the link graph data store.
	linkIt, err := svc.config.GraphAPI.Links(fromID, toID, retrievedBefore)
	if err != nil {
		return err
	}

	// Iterate and load the links into the [bsp graph] graph associated with the service's
	// calculator instance.
	for linkIt.Next() {
		svc.calculator.AddVertex(linkIt.Link().ID.String())
	}

	// Check for iteration errors.
	if err := linkIt.Error(); err != nil {
		_ = linkIt.Close()

		return err
	}

	return linkIt.Close()
}

func (svc *Service) loadEdges(fromID, toID uuid.UUID, updatedBefore time.Time) error {
	// Retrieve edges iterator for the queried partition from the link graph data store.
	edgeIt, err := svc.config.GraphAPI.Edges(fromID, toID, updatedBefore)
	if err != nil {
		return err
	}

	// Iterate and load the edges into the [bsp graph] graph associated with the service's
	// calculator instance.
	for edgeIt.Next() {
		e := edgeIt.Edge()
		err := svc.calculator.AddEdge(e.Src.String(), e.Dest.String())
		if err != nil {
			if !errors.Is(err, bsp.ErrUnknownEdgeSource) {
				_ = edgeIt.Close()

				return err
			}
		}
	}

	// Check for iteration errors.
	if err := edgeIt.Error(); err != nil {
		_ = edgeIt.Close()

		return err
	}

	return edgeIt.Close()
}
