package metricsprometheus

import (
	"context"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/command"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/prometheus/client_golang/prometheus"
)

type commandSinkMetrics struct {
	next       stream.CommandSinker
	counterOk  *prometheus.CounterVec
	counterBad *prometheus.CounterVec
	histogram  *prometheus.HistogramVec
}

func NewCommandSinkerMetrics(r prometheus.Registerer) stream.CommandSinkerInterceptor {
	return func(sinker stream.CommandSinker) stream.CommandSinker {
		wrapper := &commandSinkMetrics{
			next: sinker,
			histogram: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "gs",
					Name:      "command_sink_requests_duration_seconds",
					Buckets:   []float64{0.01, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.7, 0.8, 0.9, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5, 10},
				},
				[]string{"stream", "command"},
			),
			counterOk: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gs",
					Name:      "command_sink_requests_total",
				},
				[]string{"stream", "command"},
			),
			counterBad: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gs",
					Name:      "command_sink_error_requests_total",
				},
				[]string{"stream", "command"},
			),
		}
		r.MustRegister(
			wrapper.counterOk,
			wrapper.counterBad,
			wrapper.histogram,
		)
		return wrapper
	}
}

func (l *commandSinkMetrics) CommandSink(ctx context.Context, cmd *command.Command) (*command.Reply, error) {
	startTime := time.Now()
	reply, err := l.next.CommandSink(ctx, cmd)
	took := time.Since(startTime)
	if err != nil {
		l.counterBad.WithLabelValues(
			cmd.StreamName(),
			cmd.Name(),
		).Inc()
	} else {
		l.counterOk.WithLabelValues(
			cmd.StreamName(),
			cmd.Name(),
		).Inc()
	}
	l.histogram.WithLabelValues(
		cmd.StreamName(),
		cmd.Name(),
	).Observe(took.Seconds())
	return reply, err
}
