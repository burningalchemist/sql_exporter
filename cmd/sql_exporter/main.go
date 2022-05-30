package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"

	_ "net/http/pprof"

	"github.com/burningalchemist/sql_exporter"
	_ "github.com/kardianos/minwinsvc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"k8s.io/klog/v2"
)

const (
	envConfigFile = "SQLEXPORTER_CONFIG"
	envDebug      = "SQLEXPORTER_DEBUG"
)

var (
	showVersion   = flag.Bool("version", false, "Print version information")
	listenAddress = flag.String("web.listen-address", ":9399", "Address to listen on for web interface and telemetry")
	metricsPath   = flag.String("web.metrics-path", "/metrics", "Path under which to expose metrics")
	enableReload  = flag.Bool("web.enable-reload", false, "Enable reload collector data handler")
	configFile    = flag.String("config.file", "sql_exporter.yml", "SQL Exporter configuration filename")
)

func init() {
	klog.InitFlags(nil)
	prometheus.MustRegister(version.NewCollector("sql_exporter"))
}

func main() {
	if os.Getenv(envDebug) != "" {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
	}

	// Override --alsologtostderr default value.
	if alsoLogToStderr := flag.Lookup("alsologtostderr"); alsoLogToStderr != nil {
		alsoLogToStderr.DefValue = "true"
		_ = alsoLogToStderr.Value.Set("true")
	}
	// Override the config.file default with the SQLEXPORTER_CONFIG environment variable if set.
	if val, ok := os.LookupEnv(envConfigFile); ok {
		*configFile = val
	}

	flag.Parse()

	if *showVersion {
		fmt.Println(version.Print("sql_exporter"))
		os.Exit(0)
	}

	klog.Infof("Starting SQL exporter %s %s", version.Info(), version.BuildContext())

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

	// Expose refresh handler to reload query collections
	if *enableReload {
		http.HandleFunc("/reload", reloadCollectors(exporter))
	}
	klog.Infof("Listening on %s", *listenAddress)
	klog.Fatal(http.ListenAndServe(*listenAddress, nil))

}

func reloadCollectors(e sql_exporter.Exporter) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		klog.Infof("Reloading the collectors...")
		config := e.Config()
		if err := config.ReloadCollectorFiles(); err != nil {
			klog.Errorf("Error reloading collector configs - %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		// FIXME: Should be t.Collectors() instead of config.Collectors
		target, err := sql_exporter.NewTarget("", "", string(config.Target.DSN), config.Collectors, nil, config.Globals)
		if err != nil {
			klog.Errorf("Error creating a new target - %v", err)
		}
		e.UpdateTarget([]sql_exporter.Target{target})

		klog.Infof("Query collectors have been successfully reloaded")
		w.WriteHeader(http.StatusNoContent)
	}
}

// LogFunc is an adapter to allow the use of any function as a promhttp.Logger. If f is a function, LogFunc(f) is a
// promhttp.Logger that calls f.
type LogFunc func(args ...interface{})

// Println implements promhttp.Logger.
func (log LogFunc) Println(args ...interface{}) {
	log(args)
}
