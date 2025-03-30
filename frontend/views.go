package frontend

import (
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// OpenCensus views for monitoring HTTP server metrics.

var (
	ServerRequestCount = &view.View{
		Name:        "fortune/frontend/server/request_count",
		Description: "Count of HTTP requests started by Method",
		TagKeys:     []tag.Key{ochttp.Method},
		Measure:     ochttp.ServerRequestCount,
		Aggregation: view.Count(),
	}
	ServerResponseCount = &view.View{
		Name:        "fortune/frontend/server/response_count",
		Description: "Server response count by status code, route and method",
		TagKeys:     []tag.Key{ochttp.StatusCode, ochttp.KeyServerRoute, ochttp.Method},
		Measure:     ochttp.ServerLatency,
		Aggregation: view.Count(),
	}
	ServerLatency = &view.View{
		Name:        "fortune/frontend/server/response_latency",
		Description: "Server response distribution by status code, route and method",
		TagKeys:     []tag.Key{ochttp.StatusCode, ochttp.KeyServerRoute, ochttp.Method},
		Measure:     ochttp.ServerLatency,
		Aggregation: ochttp.DefaultLatencyDistribution,
	}
	ServerResponseBytes = &view.View{
		Name:        "fortune/frontend/server/response_bytes",
		Description: "Size distribution of HTTP response body",
		TagKeys:     []tag.Key{ochttp.StatusCode, ochttp.KeyServerRoute, ochttp.Method},
		Measure:     ochttp.ServerResponseBytes,
		Aggregation: ochttp.DefaultSizeDistribution,
	}
	ServerViews = []*view.View{
		ServerRequestCount,
		ServerResponseCount,
		ServerLatency,
		ServerResponseBytes,
	}
)
