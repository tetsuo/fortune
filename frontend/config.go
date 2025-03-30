package frontend

import (
	"fmt"
	"os"
	"time"

	"github.com/tetsuo/fortune/internal/database"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	// Name of the application.
	Name string `json:"name"`

	// Unique identifier for the current version of the application.
	VersionID string `json:"versionId"`

	// Git commit hash of the current application version.
	VersionCommitHash string `json:"versionCommitHash"`

	// Git commit date of the current application version.
	VersionCommitDate string `json:"versionCommitDate"`

	// Hostname of the machine or container where the application is running.
	Hostname string `env:"HOSTNAME" json:"hostname"`

	// GCP configuration.
	// These values are set to dummy values in locally.

	// The Google Cloud project ID associated with this application.
	ProjectID string `json:"projectId"`

	// The GCP zone where the application is deployed.
	ZoneID string `json:"zoneId"`

	// The unique instance ID of the VM or container running this application.
	InstanceID string `json:"instanceId"`

	// The service account being used by the application to authenticate with GCP services.
	ServiceAccount string `json:"serviceAccount"`

	// Name of the container running the application.
	ContainerName string `env:"CONTAINER_NAME" json:"containerName"`

	// Name of the Kubernetes cluster where the application is deployed.
	ClusterName string `env:"CLUSTER_NAME" json:"clusterName"`

	// Namespace in which the application is running within a Kubernetes cluster.
	// This can also be retrieved from /var/run/secrets/kubernetes.io/serviceaccount/namespace.
	NamespaceName string `env:"NAMESPACE_NAME" json:"namespaceName"`

	// The main HTTP port on which the application listens for requests.
	Port int `env:"PORT" envDefault:"8080" json:"port"`

	// The port used for debugging (e.g., pprof, metrics, or other debug endpoints).
	DebugPort int `env:"DEBUG_PORT" envDefault:"8081" json:"debugPort"`

	// The logging level of the application (e.g., "info", "debug", "warn", "error").
	LogLevel string `env:"LOG_LEVEL" envDefault:"info" json:"logLevel"`

	// Flags for enabling/disabling various debug endpoints and servers.

	// If true, disables the debug server.
	DisableDebugServer bool `env:"DISABLE_DEBUG_SERVER" json:"disableDebugServer"`

	// If true, disables the pprof debugging endpoint.
	DisablePProfEndpoint bool `env:"DISABLE_DEBUG_SERVER_PPROF" json:"disablePprofEndpoint"`

	// If true, disables the Prometheus metrics endpoint.
	DisablePrometheusEndpoint bool `env:"DISABLE_DEBUG_SERVER_STATS" json:"disablePrometheusEndpoint"`

	// If true, disables error reporting; locally this is always disabled.
	DisableErrorReporting bool `env:"DISABLE_ERROR_REPORTING" json:"disableErrorReporting"`

	// Flags for enabling/disabling profiling and tracing.

	// If true, enables CPU and memory profiling.
	UseProfiler bool `env:"USE_PROFILER" json:"useProfiler"`

	// If true, enables request tracing.
	UseTracer bool `env:"USE_TRACER" json:"useTracer"`

	// Maximum buffer size for storing trace spans before sending them.
	// Default: 32 MB (33554432 bytes).
	TraceSpansBufferMaxBytes int `env:"TRACE_SPANS_BUFFER_MAX_BYTES" envDefault:"33554432" json:"traceSpansBufferMaxBytes"`

	// The fraction of requests to sample for tracing.
	// Default: 0.01 (1% of requests will be traced).
	TraceSamplerFraction float64 `env:"TRACE_SAMPLER_FRACTION" envDefault:"0.01" json:"traceSamplerFraction"`

	// View (metrics) reporting period for Stackdriver monitoring.
	// Due to Stackdriver limitations, statistics should be reported at least every minute.
	// Default: 1 minute.
	ViewReportingPeriod time.Duration `env:"VIEW_REPORTING_PERIOD" envDefault:"1m" json:"viewReportingPeriod"`

	// Kubernetes service port (if running in a Kubernetes environment).
	// This value is usually set by Kubernetes.
	KubernetesServicePort int `env:"KUBERNETES_SERVICE_PORT" envDefault:"0" json:"-"`

	// Database configuration settings.
	DB database.DBConfig
}

func (c Config) IsRunningOnGCE() bool {
	return c.KubernetesServicePort != 0 && !c.IsRunningOnKind()
}

func (c Config) IsRunningOnKind() bool {
	return os.Getenv("CLUSTER_ENV") == "kind"
}

func (c Config) MustParseZapLogLevel() zapcore.Level {
	level, err := zapcore.ParseLevel(c.LogLevel)
	if err != nil {
		panic(fmt.Errorf("zapcore.ParseLevel(%s): %v", c.LogLevel, err))
	}
	return level
}

func (c Config) PodName() string {
	if c.IsRunningOnKind() || c.IsRunningOnGCE() {
		return os.Getenv("HOSTNAME")
	} else {
		return "localhost"
	}
}

func (c Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Hostname, c.Port)
}

func (c Config) DebugServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Hostname, c.DebugPort)
}

func (c Config) ZapLogLevel() zapcore.Level {
	switch c.LogLevel {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.FatalLevel
	}
}
