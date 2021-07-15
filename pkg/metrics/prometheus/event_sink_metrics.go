package metricsprometheus

import (
	"context"
	"time"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/prometheus/client_golang/prometheus"
)

type eventSinkMetrics struct {
	next       stream.EventSinker
	counterOk  *prometheus.CounterVec
	counterBad *prometheus.CounterVec
	histogram  *prometheus.HistogramVec
}

func NewEventSinkerMetrics(r prometheus.Registerer) stream.EventSinkerInterceptor {
	return func(sinker stream.EventSinker) stream.EventSinker {
		wrapper := &eventSinkMetrics{
			next: sinker,
			histogram: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "gs",
					Name:      "event_sink_requests_duration_seconds",
					Buckets:   []float64{0.01, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.7, 0.8, 0.9, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5, 10},
				},
				[]string{"stream", "event"},
			),
			counterOk: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gs",
					Name:      "event_sink_requests_total",
				},
				[]string{"stream", "event"},
			),
			counterBad: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gs",
					Name:      "event_sink_error_requests_total",
				},
				[]string{"stream", "event"},
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

func (l *eventSinkMetrics) EventSink(ctx context.Context, e *event.Event) error {
	startTime := time.Now()
	err := l.next.EventSink(ctx, e)
	took := time.Since(startTime)
	if err != nil {
		l.counterBad.WithLabelValues(
			e.StreamName(),
			e.Name(),
		).Inc()
	} else {
		l.counterOk.WithLabelValues(
			e.StreamName(),
			e.Name(),
		).Inc()
	}
	l.histogram.WithLabelValues(
		e.StreamName(),
		e.Name(),
	).Observe(took.Seconds())
	return err
}
