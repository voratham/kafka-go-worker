package handler

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	kafka_go_worker "kafka-go-worker"
	"time"
)

type consumerLabel struct {
	isError      bool
	duration     time.Duration
	messageCount int
}

func withPrometheusMetric(prefix string, prom *prometheus.Registry, config kafka_go_worker.WorkerConfig) Handler {
	labels := prometheus.Labels{
		"worker_name": config.WorkerName,
		"topic":       config.TopicName,
	}

	messagesTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:        fmt.Sprintf("%s_total", prefix),
		Help:        "Count of all Messages",
		ConstLabels: labels,
	}, []string{"is_error"})

	processDuration := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:        fmt.Sprintf("%s_duration_seconds", prefix),
		Help:        "Duration of message being processed",
		ConstLabels: labels,
	}, []string{"is_error"})

	prom.MustRegister(messagesTotal, processDuration)

	chRequestFinish := make(chan consumerLabel, config.BatchSize)

	go func() {
		for {
			select {
			case label := <-chRequestFinish:
				messagesTotal.With(prometheus.Labels{
					"is_error": fmt.Sprintf("%t", label.isError),
				}).Add(float64(label.messageCount))
				processDuration.With(prometheus.Labels{
					"is_error": fmt.Sprintf("%t", label.isError),
				}).Observe(label.duration.Seconds())
			}
		}
	}()

	return func(c *Context) error {
		start := time.Now()
		msgCount := len(c.Messages)

		err := c.Next()

		chRequestFinish <- consumerLabel{
			isError:      err != nil,
			messageCount: msgCount,
			duration:     time.Since(start),
		}

		return err
	}
}