package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/burningalchemist/sql_exporter"
	cfg "github.com/burningalchemist/sql_exporter/config"
	"github.com/go-kit/log"
	_ "github.com/kardianos/minwinsvc"
	"github.com/prometheus/client_golang/prometheus"
	info "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"k8s.io/klog/v2"
)

const (
	appName string = "sql_exporter"

	httpReadHeaderTimeout time.Duration = time.Duration(time.Second * 60)
	debugMaxLevel         klog.Level    = 3
)

var (
	showVersion   = flag.Bool("version", false, "Print version information")
	listenAddress = flag.String("web.listen-address", ":9399", "Address to listen on for web interface and telemetry")
	metricsPath   = flag.String("web.metrics-path", "/metrics", "Path under which to expose metrics")
	enableReload  = flag.Bool("web.enable-reload", false, "Enable reload collector data handler")
	webConfigFile = flag.String("web.config.file", "", "[EXPERIMENTAL] TLS/BasicAuth configuration file path")
	configFile    = flag.String("config.file", "sql_exporter.yml", "SQL Exporter configuration file path")
	logFormatJSON = flag.Bool("log.json", false, "[DEPRECATED] Set log output format to JSON")
	logFormat     = flag.String("log.format", "logfmt", "Set log output format")
	logLevel      = flag.String("log.level", "info", "Set log level")
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
	logger, err := setupLogging(*logLevel, *logFormat, *logFormatJSON)
	if err != nil {
		fmt.Printf("Error initializing exporter: %s\n", err)
		os.Exit(1)
	}

	// Override the config.file default with the SQLEXPORTER_CONFIG environment variable if set.
	if val, ok := os.LookupEnv(cfg.EnvConfigFile); ok {
		*configFile = val
	}

	klog.Warningf("Starting SQL exporter %s %s", version.Info(), version.BuildContext())
	exporter, err := sql_exporter.NewExporter(*configFile)
	if err != nil {
		klog.Fatalf("Error creating exporter: %s", err)
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
	}, logger); err != nil {
		klog.Fatal(err)
	}
}

// reloadHandler returns a handler that reloads collector and target data.
func reloadHandler(e sql_exporter.Exporter, configFile string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := sql_exporter.Reload(e, &configFile); err != nil {
			klog.Error(err)
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
				klog.Error(err)
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
	klog.Warning("Started scrape_errors_total metrics drop ticker: ", interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			exporter.DropErrorMetrics()
		}
	}()
}

// setupLogging configures and initializes the logging system.
func setupLogging(logLevel, logFormat string, logFormatJSON bool) (log.Logger, error) {
	promlogConfig := &promlog.Config{
		Level:  &promlog.AllowedLevel{},
		Format: &promlog.AllowedFormat{},
	}

	if err := promlogConfig.Level.Set(logLevel); err != nil {
		return nil, err
	}

	// Override log format if JSON is specified.
	finalLogFormat := logFormat
	if logFormatJSON {
		fmt.Print("Warning: The flag --log.json is deprecated and will be removed in a future release. Please use --log.format=json instead\n")
		finalLogFormat = "json"
	}
	if err := promlogConfig.Format.Set(finalLogFormat); err != nil {
		return nil, err
	}
	// Overriding the default klog with our go-kit klog implementation.
	logger := promlog.New(promlogConfig)
	klog.SetLogger(logger)
	klog.ClampLevel(debugMaxLevel)

	return logger, nil
}
