package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	linkgraphapi "github.com/mycok/uSearch/linkgraph/store/api/rpc"
	linkgraphproto "github.com/mycok/uSearch/linkgraph/store/api/rpc/graphproto"
	"github.com/mycok/uSearch/monolith/partition"
	"github.com/mycok/uSearch/monolith/service/pagerank"
	textindexerapi "github.com/mycok/uSearch/textindexer/store/api/rpc"
	textindexerproto "github.com/mycok/uSearch/textindexer/store/api/rpc/indexproto"
)

var (
	appName = "usearch-pagerank"
	appSHA  = "latest-app-git-sha" // Populated by the compiler at the linking stage.
	logger  *logrus.Entry
)

func main() {
	host, _ := os.Hostname()
	rootLogger := logrus.New()
	rootLogger.SetFormatter(new(logrus.JSONFormatter))
	logger = rootLogger.WithFields(logrus.Fields{
		"app":  appName,
		"sha":  appSHA,
		"host": host,
	})

	if err := configureAppEnv().Run(os.Args); err != nil {
		logger.WithField("err", err).Error("shutting down due to an error")
		_ = os.Stderr.Sync()

		os.Exit(1)
	}
}

func configureAppEnv() *cli.App {
	app := cli.NewApp()
	app.Name = appName
	app.Version = appSHA
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "link-graph-api",
			EnvVars: []string{"LINK_GRAPH_API"},
			Usage:   "gRPC endpoint for connecting to the link graph service",
		},
		&cli.StringFlag{
			Name:    "text-indexer-api",
			EnvVars: []string{"TEXT_INDEXER_API"},
			Usage:   "gRPC endpoint for connecting to the text indexer service",
		},
		&cli.IntFlag{
			Name:    "num-of-workers",
			Value:   runtime.NumCPU(),
			EnvVars: []string{"NUM_OF_WORKERS"},
			Usage:   "Number of workers to use for calculating page-rank scores",
		},
		&cli.DurationFlag{
			Name:    "update-interval",
			Value:   5 * time.Minute,
			EnvVars: []string{"PAGERANK_UPDATE_INTERVAL"},
			Usage:   "Time between subsequent page-rank runs",
		},
		&cli.StringFlag{
			Name:    "partition-detection-mode",
			Value:   "single",
			EnvVars: []string{"PARTITION_DETECTION_MODE"},
			Usage:   "The partition detection mode to use. Supported values are 'dns=HEADLESS_SERVICE_NAME' for (k8s) and 'single' for (local dev mode)",
		},
		&cli.IntFlag{
			Name:    "pprof-port",
			Value:   6060,
			EnvVars: []string{"PPROF_PORT"},
			Usage:   "port for exposing pprof endpoints",
		},
	}

	app.Action = execute

	return app
}

func execute(appCtx *cli.Context) error {
	var wg sync.WaitGroup

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	// Configure page-rank.
	partDet, err := getPartitionDetector(appCtx.String("partition-detection-mode"))
	if err != nil {
		return err
	}

	graphAPI, indexAPI, err := getAPIs(
		ctx, appCtx.String("link-graph-api"), appCtx.String("text-indexer-api"),
	)
	if err != nil {
		return err
	}

	var config pagerank.Config
	config.GraphAPI = graphAPI
	config.IndexAPI = indexAPI
	config.NumOfComputeWorkers = appCtx.Int("num-of-workers")
	config.UpdateInterval = appCtx.Duration("update-interval")
	config.PartitionDetector = partDet
	config.Logger = logger

	pagerankSvc, err := pagerank.New(config)
	if err != nil {
		return err
	}

	// Run crawler.
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := pagerankSvc.Run(ctx); err != nil {
			logger.WithField("err", err).Error("page-rank service exited with an error")

			cancelFn()
		}
	}()

	// Start pprof server.
	pprofListener, err := net.Listen("tcp", fmt.Sprintf(":%d", appCtx.Int("pprof-port")))
	if err != nil {
		return err
	}
	defer func() { _ = pprofListener.Close() }()

	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.WithField("port", appCtx.Int("pprof-port")).Info("listening for pprof requests")

		srv := new(http.Server)
		_ = srv.Serve(pprofListener)
	}()

	// Start os signal watcher.
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGHUP)

		select {
		case s := <-signalChan:
			logger.WithField("signal", s.String()).Info("shutting down due to signal")

			// Closing the pprof listener causes the pprof server to return / exit.
			_ = pprofListener.Close()

			cancelFn()

		case <-ctx.Done():
			// Closing the pprof listener causes the pprof server to return / exit.
			_ = pprofListener.Close()

		}
	}()

	wg.Wait()

	return nil
}

func getPartitionDetector(mode string) (partition.Detector, error) {
	switch {
	case mode == "single":
		return partition.DummyDetector{
			Partition:       0,
			NumOfPartitions: 1,
		}, nil
	case strings.HasPrefix(mode, "dns="):
		tokens := strings.Split(mode, "=")
		return partition.DetectFromSRVRecords(tokens[1]), nil
	default:
		return nil, fmt.Errorf("unsupported partition detector mode: %q", mode)
	}
}

func getAPIs(
	ctx context.Context, linkGraphAPI, textIndexerAPI string,
) (*linkgraphapi.LinkGraphClient, *textindexerapi.TextIndexerClient, error) {

	if linkGraphAPI == "" {
		return nil, nil, fmt.Errorf("link graph API must be specified with --link-graph-api")
	}

	if textIndexerAPI == "" {
		return nil, nil, fmt.Errorf("text indexer API must be specified with --text-indexer-api")
	}

	dialCtx, cancelFn := context.WithTimeout(ctx, 5*time.Second)
	defer cancelFn()
	GraphConn, err := grpc.DialContext(
		dialCtx, linkGraphAPI,
		grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to link graph API: %w", err)
	}
	graphClient := linkgraphapi.NewLinkGraphClient(ctx, linkgraphproto.NewLinkGraphClient(GraphConn))

	dialCtx, cancelFn = context.WithTimeout(ctx, 5*time.Second)
	defer cancelFn()
	indexConn, err := grpc.DialContext(
		dialCtx, textIndexerAPI,
		grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to text indexer API: %w", err)
	}
	IndexClient := textindexerapi.NewTextIndexerClient(ctx, textindexerproto.NewTextIndexerClient(indexConn))

	return graphClient, IndexClient, nil
}
