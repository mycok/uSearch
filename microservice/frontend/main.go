package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	linkgraphapi "github.com/mycok/uSearch/linkgraph/store/api/rpc"
	linkgraphproto "github.com/mycok/uSearch/linkgraph/store/api/rpc/graphproto"
	"github.com/mycok/uSearch/monolith/service/frontend"
	textindexerapi "github.com/mycok/uSearch/textindexer/store/api/rpc"
	textindexerproto "github.com/mycok/uSearch/textindexer/store/api/rpc/indexproto"
)

var (
	appName = "usearch-frontend"
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
			Usage:   "gRPC endpoint for connecting to the text indexer service ",
		},
		&cli.IntFlag{
			Name:    "frontend-port",
			Value:   8080,
			EnvVars: []string{"FRONTEND_PORT"},
			Usage:   "Exposed port for the frontend",
		},
		&cli.IntFlag{
			Name:    "num-results-per-page",
			Value:   10,
			EnvVars: []string{"NUM_OF_RESULTS_PER_PAGE"},
			Usage:   "Max number of displayed results per page",
		},
		&cli.IntFlag{
			Name:    "max-summary-length",
			Value:   256,
			EnvVars: []string{"MAX_SUMMARY_LENGTH"},
			Usage:   "The maximum length (in characters) of page content summary for each matched document",
		},
		&cli.IntFlag{
			Name:    "pprof-port",
			Value:   6060,
			EnvVars: []string{"PPROF_PORT"},
			Usage:   "Exposed port for pprof endpoints",
		},
	}

	app.Action = execute

	return app
}

func execute(appCtx *cli.Context) error {
	var wg sync.WaitGroup

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	graphAPI, indexAPI, err := getAPIs(
		ctx, appCtx.String("link-graph-api"), appCtx.String("text-indexer-api"),
	)
	if err != nil {
		return err
	}

	var config frontend.Config
	config.GraphAPI = graphAPI
	config.IndexAPI = indexAPI
	config.ListenAddr = fmt.Sprintf(":%d", appCtx.Int("frontend-port"))
	config.NumOfResultsPerPage = appCtx.Int("num-results-per-page")
	config.MaxSummaryLength = appCtx.Int("max-summary-length")
	config.Logger = logger

	frontendSvc, err := frontend.New(config)
	if err != nil {
		return err
	}

	// Run crawler.
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := frontendSvc.Run(ctx); err != nil {
			logger.WithField("err", err).Error("frontend service exited with an error")

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
