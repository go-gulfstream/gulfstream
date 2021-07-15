package metricsprometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

type eventHandlerMetric struct {
	next               stream.EventHandler
	counterMatch       *prometheus.CounterVec
	counterHandleOk    *prometheus.CounterVec
	counterHandleBad   *prometheus.CounterVec
	histogramHandle    *prometheus.HistogramVec
	counterRollbackOk  *prometheus.CounterVec
	counterRollbackBad *prometheus.CounterVec
	histogramRollback  *prometheus.HistogramVec
}

func NewEventHandlerMetrics(r prometheus.Registerer) stream.EventHandlerInterceptor {
	return func(handler stream.EventHandler) stream.EventHandler {
		wrapper := &eventHandlerMetric{
			next: handler,
			counterMatch: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gs",
					Name:      "event_handler_match_total",
				},
				[]string{"event"},
			),
			counterHandleOk: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gs",
					Name:      "event_handler_receiver_total",
				},
				[]string{"stream", "event"},
			),
			counterHandleBad: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gs",
					Name:      "event_handler_error_receiver_total",
				},
				[]string{"stream", "event"},
			),
			histogramHandle: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "gs",
					Name:      "event_handler_receiver_duration_seconds",
					Buckets:   []float64{0.01, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.7, 0.8, 0.9, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5, 10},
				},
				[]string{"stream", "event"},
			),
			counterRollbackOk: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gs",
					Name:      "event_handler_rollback_receiver_total",
				},
				[]string{"stream", "event"},
			),
			counterRollbackBad: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gs",
					Name:      "event_handler_rollback_error_receiver_total",
				},
				[]string{"stream", "event"},
			),
			histogramRollback: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "gs",
					Name:      "event_handler_rollback_receiver_duration_seconds",
					Buckets:   []float64{0.01, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.7, 0.8, 0.9, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5, 10},
				},
				[]string{"stream", "event"},
			),
		}
		r.MustRegister(
			wrapper.counterMatch,
			wrapper.counterHandleOk,
			wrapper.counterHandleBad,
			wrapper.histogramHandle,
			wrapper.counterRollbackBad,
			wrapper.counterRollbackOk,
			wrapper.histogramRollback,
		)
		return wrapper
	}
}

func (h *eventHandlerMetric) Match(eventName string) bool {
	h.counterMatch.WithLabelValues(eventName).Inc()
	return h.next.Match(eventName)
}

func (h *eventHandlerMetric) Handle(ctx context.Context, e *event.Event) error {
	startTime := time.Now()
	err := h.next.Handle(ctx, e)
	took := time.Since(startTime)
	if err != nil {
		h.counterHandleBad.WithLabelValues(e.StreamName(), e.Name()).Inc()
	} else {
		h.counterHandleOk.WithLabelValues(e.StreamName(), e.Name()).Inc()
	}
	h.histogramHandle.WithLabelValues(e.StreamName(), e.Name()).Observe(took.Seconds())
	return err
}

func (h *eventHandlerMetric) Rollback(ctx context.Context, e *event.Event) error {
	startTime := time.Now()
	err := h.next.Rollback(ctx, e)
	took := time.Since(startTime)
	if err != nil {
		h.counterRollbackBad.WithLabelValues(e.StreamName(), e.Name()).Inc()
	} else {
		h.counterRollbackOk.WithLabelValues(e.StreamName(), e.Name()).Inc()
	}
	h.histogramRollback.WithLabelValues(e.StreamName(), e.Name()).Observe(took.Seconds())
	return err
}
