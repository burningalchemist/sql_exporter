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
	_ "github.com/kardianos/minwinsvc"
	"github.com/prometheus/client_golang/prometheus"
	info "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	logFormatJSON = flag.Bool("log.json", false, "Set log output format to JSON")
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
	promlogConfig := &promlog.Config{}
	promlogConfig.Level = &promlog.AllowedLevel{}
	err := promlogConfig.Level.Set(*logLevel)
	if err != nil {
		fmt.Printf("Error initializing exporter: %s\n", err)
		os.Exit(1)
	}
	if *logFormatJSON {
		promlogConfig.Format = &promlog.AllowedFormat{}
		_ = promlogConfig.Format.Set("json")
	}

	// Overriding the default klog with our go-kit klog implementation.
	// Thus we need to pass it our go-kit logger object.
	logger := promlog.New(promlogConfig)
	klog.SetLogger(logger)
	klog.ClampLevel(debugMaxLevel)

	// Override the config.file default with the SQLEXPORTER_CONFIG environment variable if set.
	if val, ok := os.LookupEnv(cfg.EnvConfigFile); ok {
		*configFile = val
	}

	klog.Warningf("Starting SQL exporter %s %s", version.Info(), version.BuildContext())

	exporter, err := sql_exporter.NewExporter(*configFile)
	if err != nil {
		klog.Fatalf("Error creating exporter: %s", err)
	}

	// Setup and start webserver.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "OK", http.StatusOK) })
	http.HandleFunc("/", HomeHandlerFunc(*metricsPath))
	http.HandleFunc("/config", ConfigHandlerFunc(*metricsPath, exporter))
	http.Handle(*metricsPath, promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, ExporterHandlerFor(exporter)))
	// Expose exporter metrics separately, for debugging purposes.
	http.Handle("/sql_exporter_metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
	// Expose refresh handler to reload collectors and targets
	if *enableReload {
		http.HandleFunc("/reload", reloadHandler(exporter))
	}

	// Drop scrape error metrics if configured
	scrapeErrorsDropInterval := exporter.Config().Globals.ScrapeErrorDropInterval
	if scrapeErrorsDropInterval > 0 {
		ticker := time.NewTicker(time.Duration(scrapeErrorsDropInterval))
		klog.Warning("Started scrape_errors_total metrics drop ticker: ", scrapeErrorsDropInterval)
		defer ticker.Stop()
		go func() {
			for range ticker.C {
				sql_exporter.DropErrorMetrics()
				klog.Info("Dropped scrape_errors_total metrics")
			}
		}()
	}

	// Handle SIGHUP for reloading the configuration
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)
		for {
			<-c
			err := sql_exporter.Reload(exporter, configFile)
			if err != nil {
				klog.Error(err)
				continue
			}
		}
	}()

	// Start the web server
	server := &http.Server{Addr: *listenAddress, ReadHeaderTimeout: httpReadHeaderTimeout}
	if err := web.ListenAndServe(server, &web.FlagConfig{
		WebListenAddresses: &([]string{*listenAddress}),
		WebConfigFile:      webConfigFile, WebSystemdSocket: OfBool(false),
	}, logger); err != nil {
		klog.Fatal(err)
	}
}

// reloadHandler returns a handler that reloads collectors and targets
func reloadHandler(e sql_exporter.Exporter) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := sql_exporter.Reload(e, configFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
