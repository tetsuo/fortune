package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/errorreporting"
	"cloud.google.com/go/profiler"
	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/tetsuo/fortune/cmd/internal/cmdconfig"
	"github.com/tetsuo/fortune/cmd/internal/dcensus"
	"github.com/tetsuo/fortune/frontend"
	"github.com/tetsuo/fortune/internal/middleware"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

var hostFlag = flag.String("host", "localhost", "")

var (
	name    = "fortune"
	version = "development"
	commit  = ""
	date    = ""
)

func main() {
	flag.Parse()

	cfg := frontendConfig(name, version, commit, date)

	devMode := !cfg.IsRunningOnGCE()

	if devMode {
		cfg.Hostname = *hostFlag
	}

	baseLogger, err := cmdconfig.NewLogger(cfg.ZapLogLevel(), devMode)
	if err != nil {
		panic(err)
	}

	log := baseLogger.Sugar()

	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		log.Fatalf("error marshaling JSON: %v", err)
	}
	cfgMap := map[string]any{}
	if err := json.Unmarshal(cfgBytes, &cfgMap); err != nil {
		log.Fatalf("error unmarshaling JSON: %v", err)
	}

	log.With("cfg", cfgMap).With("devMode", devMode).Info("configured")

	var wg sync.WaitGroup

	wg.Add(1)

	// Add exit handlers

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func(sigs chan os.Signal) {
		sig := <-sigs

		close(sigs)

		log.Infof("interrupted by signal %s", sig.String())

		time.AfterFunc(time.Minute*1, func() {
			log.Fatal("exiting forcefully after waiting 60 seconds")
		})

		wg.Done()
	}(sigs)

	shutdown := func() {
		defer func() {
			_ = recover()
		}()
		sigs <- syscall.SIGINT
	}

	// debug server enabled?
	var debugServer *http.Server
	if !cfg.DisableDebugServer {
		var pe *prometheus.Exporter
		if !cfg.DisablePrometheusEndpoint {
			log.Info("enabling metrics")
			view.SetReportingPeriod(cfg.ViewReportingPeriod)
			pe, err = prometheus.NewExporter(prometheus.Options{})
			if err != nil {
				log.Fatalf("error initializing prometheus exporter: %v", err)
			}
			if err := view.Register(frontend.ServerViews...); err != nil {
				log.Fatalf("error registering frontend server views: %v", err)
			}
		}
		debugHandler, err := dcensus.ServeDebug(pe, !cfg.DisablePProfEndpoint, version, cfg.PodName(), cfg.Name, cfg.VersionCommitHash, cfg.VersionCommitDate)
		if err != nil {
			log.Fatalf("error initializing debug server: %v", err)
		}
		debugServer = &http.Server{Addr: cfg.DebugServerAddress(), Handler: debugHandler}

		go func(s *http.Server) {
			log.Infof("debug server listening on %s", cfg.DebugServerAddress())
			log.Errorf("debug server exited: %v", s.ListenAndServe())
			shutdown()
		}(debugServer)
	}

	// use gcp profiler?
	if !devMode && cfg.UseProfiler {
		log.Info("enabling profiler")
		if err := profiler.Start(profiler.Config{
			Service:        cfg.ContainerName,
			ServiceVersion: cfg.VersionID,
		}); err != nil {
			log.Fatalf("error starting profiler: %v", err)
		}
	}

	var er *errorreporting.Client

	// report to gcp error reporting service?
	if !devMode && !cfg.DisableErrorReporting {
		log.Info("enabling error reporting")
		er, err = errorreporting.NewClient(context.Background(), cfg.ProjectID, errorreporting.Config{
			ServiceName:    fmt.Sprintf("%s/frontend", name),
			ServiceVersion: version,
			OnError: func(err error) {
				log.Errorf("error reporting error: %v", err)
			},
		})
		if err != nil {
			log.Fatalf("error initializing errorreporting client: %v", err)
		}
	}

	// export traces to stackdriver?
	if !devMode && cfg.UseTracer {
		log.With("traceSamplerFraction", cfg.TraceSamplerFraction, "traceSpansMaxBufferBytes", cfg.TraceSpansBufferMaxBytes).
			Info("enabling tracer")
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(cfg.TraceSamplerFraction)})
		dcensus.RegisterTraceExporter(
			dcensus.NewMonitoredResource(cfg.ProjectID, cfg.ZoneID, cfg.ClusterName, cfg.NamespaceName, cfg.ContainerName, cfg.PodName(), cfg.VersionID),
			cfg.VersionID,
			cfg.ProjectID,
			cfg.TraceSpansBufferMaxBytes,
		)
	}

	db, err := cmdconfig.OpenDB(context.Background(), cfg.InstanceID, cfg.DB)
	if err != nil {
		log.Fatalf("error opening DB: %v", err)
	}

	s, err := frontend.NewServer(cfg, db, er)
	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}

	errorReportingMiddleware := middleware.Empty()
	if er != nil {
		errorReportingMiddleware = middleware.ErrorReporting(er.Report)
	}

	r := dcensus.NewRouter(nil)
	s.Install(r.Handle)

	server := &http.Server{
		Addr: cfg.ServerAddress(),
		Handler: middleware.Chain(
			middleware.AcceptRequests(http.MethodGet, http.MethodPost),
			middleware.Panic(s.PanicHandler()),
			errorReportingMiddleware,
			middleware.Timeout(54*time.Second),
		)(r),
	}

	go func(s *http.Server, serverAddress string) {
		log.Infof("frontend server listening on %s", serverAddress)
		log.Errorf("frontend server exited: %v", s.ListenAndServe())
		shutdown()
	}(server, cfg.ServerAddress())

	wg.Wait()

	log.Infof("shutting down frontend server on %s", cfg.ServerAddress())
	if err := server.Shutdown(context.Background()); err != nil {
		log.Errorf("error shutting down frontend server: %v", err)
	}

	if debugServer != nil {
		log.Infof("shutting down debug server on %s", cfg.DebugServerAddress())
		if err := debugServer.Shutdown(context.Background()); err != nil {
			log.Errorf("error shutting down debug server: %v", err)
		}
	}

	log.Infof("disconnecting from db")
	if err := db.Close(); err != nil {
		log.Errorf("error disconnecting from db: %v", err)
	}

	if er != nil {
		if err := er.Close(); err != nil {
			log.Errorf("error closing error reporting client: %v", err)
		}
	}

	log.Info("exiting gracefully")
}
