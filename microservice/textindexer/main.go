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
	"strings"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"

	"github.com/mycok/uSearch/textindexer/index"
	textindexerapi "github.com/mycok/uSearch/textindexer/store/api/rpc"
	textindexerproto "github.com/mycok/uSearch/textindexer/store/api/rpc/indexproto"
	"github.com/mycok/uSearch/textindexer/store/es"
	"github.com/mycok/uSearch/textindexer/store/memory"
)

var (
	appName = "usearch-textindexer"
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
			Name:    "text-index-uri",
			EnvVars: []string{"TEXT_INDEX_URI"},
			Usage:   "URI for connecting to textindexer data store (supported URI's: in-memory://, es://node1:9200,...,nodeN:9200)",
		},

		&cli.IntFlag{
			Name:    "grpc-port",
			Value:   8080,
			EnvVars: []string{"GRPC_PORT"},
			Usage:   "Exposed port for text index gRPC endpoints",
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

	index, err := getTextIndexer(ctx, appCtx.String("text-index-uri"))
	if err != nil {
		return err
	}

	// Create a listener.
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", appCtx.Int("grpc-port")))
	if err != nil {
		return err
	}
	defer func() { _ = grpcListener.Close() }()

	// Start textindexer server.
	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Logger.WithField("port", appCtx.Int("grpc-port")).Info("listening for gRPC connections")

		srv := grpc.NewServer()
		textindexerproto.RegisterTextIndexerServer(srv, textindexerapi.NewTextIndexerServer(index))
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

func getTextIndexer(_ context.Context, textIndexerURI string) (index.Indexer, error) {
	if textIndexerURI == "" {
		return nil, fmt.Errorf("text indexer URI must be specified with --text-indexer-uri")
	}

	url, err := url.Parse(textIndexerURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse link graph URI: %w", err)
	}

	switch url.Scheme {
	case "in-memory":
		logger.Info("using in-memory index")

		return memory.NewInMemoryIndex()
	case "es":
		nodes := strings.Split(url.Host, ",")
		for i := 0; i < len(nodes); i++ {
			nodes[i] = "http://" + nodes[i]
		}

		logger.Info("using ES index")

		return es.NewEsIndexer(nodes, false)
	default:
		return nil, fmt.Errorf("unsupported link graph URI scheme: %q", url.Scheme)
	}
}
