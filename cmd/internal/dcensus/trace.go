package dcensus

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"sync"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"google.golang.org/genproto/googleapis/api/monitoredres"
)

type debugTraceExporter struct {
	exp trace.Exporter
	mu  sync.Mutex
	err error
}

func (d *debugTraceExporter) onError(err error) {
	zap.S().Debugf("trace exporter: onError called with %v", err)
	d.err = err
}

// ExportSpan implements the trace.Exporter interface.
func (d *debugTraceExporter) ExportSpan(s *trace.SpanData) {
	d.mu.Lock()
	if d.exp != nil {
		d.exp.ExportSpan(s)
	}
	err := d.err
	d.err = nil
	d.mu.Unlock()
	if err != nil {
		zap.S().Warnf("trace exporter: %v", err)
		zap.S().Debugf("trace exporter SpanData:\n%s", dumpSpanData(s))
	}
}

func dumpSpanData(s *trace.SpanData) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Name: %q\n", s.Name)
	dumpAttributes(&buf, s.Attributes)
	for _, a := range s.Annotations {
		fmt.Fprintf(&buf, "  annotation: %q\n", a.Message)
		dumpAttributes(&buf, a.Attributes)
	}
	fmt.Fprintf(&buf, "Status.Message: %q\n", s.Status.Message)
	fmt.Fprintln(&buf, "link attrs:")
	for _, l := range s.Links {
		dumpAttributes(&buf, l.Attributes)
	}
	return buf.String()
}

func dumpAttributes(w io.Writer, m map[string]any) {
	for k, v := range m {
		fmt.Fprintf(w, "  %q: %#v\n", k, v)
	}
}

type MonitoredResource monitoredres.MonitoredResource

func (r *MonitoredResource) MonitoredResource() (resType string, labels map[string]string) {
	return r.Type, r.Labels
}

func NewMonitoredResource(projectID, zoneID, clusterName, namespaceName, containerName, podName, versionID string) *MonitoredResource {
	return &MonitoredResource{
		Type: "k8s_container",
		Labels: map[string]string{
			"project_id":     projectID,
			"location":       path.Base(zoneID),
			"cluster_name":   clusterName,
			"namespace_name": namespaceName,
			"pod_name":       podName,
			"container_name": containerName,
			"version":        versionID,
		},
	}
}

func RegisterDebugExporter() {
	dte := &debugTraceExporter{}
	trace.RegisterExporter(dte)
}

func RegisterTraceExporter(r *MonitoredResource, versionID, projectID string, traceSpansBufferMaxBytes int) {
	dte := &debugTraceExporter{}
	labels := &stackdriver.Labels{}
	labels.Set("version", versionID, "")
	traceExporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:                projectID,
		MonitoredResource:        r,
		TraceSpansBufferMaxBytes: traceSpansBufferMaxBytes,
		DefaultMonitoringLabels:  labels,
		OnError:                  dte.onError,
	})
	if err != nil {
		zap.S().Fatalf("stackdriver.NewExporter(%q): %v", projectID, err)
	}
	dte.exp = traceExporter
	trace.RegisterExporter(dte)
}
