package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"

	"github.com/mycok/uSearch/linkgraph/graph"
	linkgraphapi "github.com/mycok/uSearch/linkgraph/store/api/rpc"
	linkgraphproto "github.com/mycok/uSearch/linkgraph/store/api/rpc/graphproto"
	"github.com/mycok/uSearch/linkgraph/store/cdb"
	"github.com/mycok/uSearch/linkgraph/store/memory"
)

var (
	appName = "usearch-linkgraph"
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
			Name:    "link-graph-uri",
			EnvVars: []string{"LINK_GRAPH_URI"},
			Usage:   "URI for connecting to the link graph data store (supported URI's: in-memory://, postgresql://user@host:26257/linkgraph?sslmode=disable",
		},

		&cli.IntFlag{
			Name:    "grpc-port",
			Value:   8080,
			EnvVars: []string{"GRPC_PORT"},
			Usage:   "Exposed port for link graph gRPC endpoints",
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

	linkGraph, err := getLinkGraph(ctx, appCtx.String("link-graph-uri"))
	if err != nil {
		return err
	}

	// Start a gRPC server.
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", appCtx.Int("grpc-port")))
	if err != nil {
		return err
	}
	defer func() { _ = grpcListener.Close() }()

	// Run linkgraph server.
	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Logger.WithField("port", appCtx.Int("grpc-port")).Info("listening for gRPC connections")

		srv := grpc.NewServer()
		linkgraphproto.RegisterLinkGraphServer(srv, linkgraphapi.NewLinkGraphServer(linkGraph))
		_ = srv.Serve(grpcListener)
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

			// Closing the pprof listener causes the gRPC server to return / exit.
			_ = grpcListener.Close()

			cancelFn()
		case <-ctx.Done():
			// Closing the pprof listener causes the pprof server to return / exit.
			_ = pprofListener.Close()

			// Closing the pprof listener causes the gRPC server to return / exit.
			_ = grpcListener.Close()
		}
	}()

	wg.Wait()

	return nil
}

func getLinkGraph(_ context.Context, linkGraphURI string) (graph.Graph, error) {
	if linkGraphURI == "" {
		return nil, fmt.Errorf("link graph URI must be specified with --link-graph-uri")
	}

	url, err := url.Parse(linkGraphURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse link graph URI: %w", err)
	}

	switch url.Scheme {
	case "in-memory":
		return memory.NewInMemoryGraph(), nil
	case "postgresql":
		return cdb.NewCockroachDBGraph(linkGraphURI)
	default:
		return nil, fmt.Errorf("unsupported link graph URI scheme: %q", url.Scheme)
	}
}
