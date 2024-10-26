package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/burningalchemist/sql_exporter"
	cfg "github.com/burningalchemist/sql_exporter/config"
	_ "github.com/kardianos/minwinsvc"
	"github.com/prometheus/client_golang/prometheus"
	info "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
)

const (
	appName string = "sql_exporter"

	httpReadHeaderTimeout time.Duration = time.Duration(time.Second * 60)
)

var (
	showVersion   = flag.Bool("version", false, "Print version information")
	listenAddress = flag.String("web.listen-address", ":9399", "Address to listen on for web interface and telemetry")
	metricsPath   = flag.String("web.metrics-path", "/metrics", "Path under which to expose metrics")
	enableReload  = flag.Bool("web.enable-reload", false, "Enable reload collector data handler")
	webConfigFile = flag.String("web.config.file", "", "[EXPERIMENTAL] TLS/BasicAuth configuration file path")
	configFile    = flag.String("config.file", "sql_exporter.yml", "SQL Exporter configuration file path")
	logFormat     = flag.String("log.format", "logfmt", "Set log output format")
	logLevel      = flag.String("log.level", "info", "Set log level")
	logFile       = flag.String("log.file", "", "Log file to write to, leave empty to write to stderr")
)

func init() {
	prometheus.MustRegister(info.NewCollector("sql_exporter"))
	flag.BoolVar(&cfg.EnablePing, "config.enable-ping", true, "Enable ping for targets")
	flag.BoolVar(&cfg.IgnoreMissingVals, "config.ignore-missing-values", false, "[EXPERIMENTAL] Ignore results with missing values for the requested columns")
	flag.StringVar(&cfg.DsnOverride, "config.data-source-name", "", "Data source name to override the value in the configuration file with")
	flag.StringVar(&cfg.TargetLabel, "config.target-label", "target", "Target label name")
}

func main() {
	if os.Getenv(cfg.EnvDebug) != "" {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
	}

	flag.Parse()

	// Show version and exit.
	if *showVersion {
		fmt.Println(version.Print(appName))
		os.Exit(0)
	}

	// Setup logging.
	logConfig, err := initLogConfig(*logLevel, *logFormat, *logFile)
	if err != nil {
		fmt.Printf("Error initializing exporter: %s\n", err)
		os.Exit(1)
	}

	defer func() {
		if logConfig.logFileHandler != nil {
			logConfig.logFileHandler.Close()
		}
	}()

	slog.SetDefault(logConfig.logger)

	// Override the config.file default with the SQLEXPORTER_CONFIG environment variable if set.
	if val, ok := os.LookupEnv(cfg.EnvConfigFile); ok {
		*configFile = val
	}

	slog.Warn("Starting SQL exporter", "versionInfo", version.Info(), "buildContext", version.BuildContext())
	exporter, err := sql_exporter.NewExporter(*configFile)
	if err != nil {
		slog.Error("Error creating exporter", "error", err)
		os.Exit(1)
	}

	// Start the scrape_errors_total metric drop ticker if configured.
	startScrapeErrorsDropTicker(exporter, exporter.Config().Globals.ScrapeErrorDropInterval)

	// Start signal handler to reload collector and target data.
	signalHandler(exporter, *configFile)

	// Setup and start webserver.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "OK", http.StatusOK) })
	http.HandleFunc("/", HomeHandlerFunc(*metricsPath))
	http.HandleFunc("/config", ConfigHandlerFunc(*metricsPath, exporter))
	http.Handle(*metricsPath, promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, ExporterHandlerFor(exporter)))
	// Expose exporter metrics separately, for debugging purposes.
	http.Handle("/sql_exporter_metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
	// Expose refresh handler to reload collectors and targets
	if *enableReload {
		http.HandleFunc("/reload", reloadHandler(exporter, *configFile))
	}

	server := &http.Server{Addr: *listenAddress, ReadHeaderTimeout: httpReadHeaderTimeout}
	if err := web.ListenAndServe(server, &web.FlagConfig{
		WebListenAddresses: &([]string{*listenAddress}),
		WebConfigFile:      webConfigFile, WebSystemdSocket: OfBool(false),
	}, logConfig.logger); err != nil {
		slog.Error("Error starting web server", "error", err)
		os.Exit(1)

	}
}

// reloadHandler returns a handler that reloads collector and target data.
func reloadHandler(e sql_exporter.Exporter, configFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := sql_exporter.Reload(e, &configFile); err != nil {
			slog.Error("Error reloading collector and target data", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// signalHandler listens for SIGHUP signals and reloads the collector and target data.
func signalHandler(e sql_exporter.Exporter, configFile string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for range c {
			if err := sql_exporter.Reload(e, &configFile); err != nil {
				slog.Error("Error reloading collector and target data", "error", err)
			}
		}
	}()
}

// startScrapeErrorsDropTicker starts a ticker that periodically drops scrape error metrics.
func startScrapeErrorsDropTicker(exporter sql_exporter.Exporter, interval model.Duration) {
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(interval))
	slog.Warn("Started scrape_errors_total metrics drop ticker", "interval", interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			exporter.DropErrorMetrics()
		}
	}()
}
