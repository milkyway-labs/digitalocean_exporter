package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// DBCollector collects metrics about all databases.
type DBCollector struct {
	logger  log.Logger
	errors  *prometheus.CounterVec
	client  *godo.Client
	timeout time.Duration

	Up *prometheus.Desc
	NumNodes *prometheus.Desc
}

// NewDBCollector returns a new DBCollector.
func NewDBCollector(logger log.Logger, errors *prometheus.CounterVec, client *godo.Client, timeout time.Duration) *DBCollector {
	errors.WithLabelValues("database").Add(0)

	labels := []string{"id", "name", "region", "type", "engine", "names"}
	return &DBCollector{
		logger:  logger,
		errors:  errors,
		client:  client,
		timeout: timeout,

		Up: prometheus.NewDesc(
			"digitalocean_db_up",
			"If 1 the db is up and running, 0 otherwise",
			labels, nil,
		),
		NumNodes: prometheus.NewDesc(
			"digitalocean_db_num_nodes",
			"Database's number of nodes",
			labels, nil,
		),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector.
func (c *DBCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Up
	ch <- c.NumNodes
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *DBCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	dbs, _, err := c.client.Databases.List(ctx, nil)
	if err != nil {
		c.errors.WithLabelValues("database").Add(1)
		level.Warn(c.logger).Log(
			"msg", "can't list dbs",
			"err", err,
		)
		return
	}

	for _, db := range dbs {
		labels := []string{
			db.ID,
			db.Name,
			db.RegionSlug,
			db.SizeSlug,
			fmt.Sprintf("%s (v%s)", db.EngineSlug, db.VersionSlug),
			strings.Join(db.DBNames, ", "),
		}

		var online float64
		if db.Status == "online" {
			online = 1.0
		}
		ch <- prometheus.MustNewConstMetric(
			c.Up,
			prometheus.GaugeValue,
			online,
			labels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.NumNodes,
			prometheus.GaugeValue,
			float64(db.NumNodes),
			labels...,
		)
	}
}
