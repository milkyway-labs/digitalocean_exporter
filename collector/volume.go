package collector

import (
	"context"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// VolumeCollector collects metrics about all volumes.
type VolumeCollector struct {
	logger  log.Logger
	errors  *prometheus.CounterVec
	client  *godo.Client
	timeout time.Duration

	Size *prometheus.Desc
}

// NewVolumeCollector returns a new VolumeCollector.
func NewVolumeCollector(logger log.Logger, errors *prometheus.CounterVec, client *godo.Client, timeout time.Duration) *VolumeCollector {
	errors.WithLabelValues("volume").Add(0)

	labels := []string{"id", "name", "region"}
	return &VolumeCollector{
		logger:  logger,
		errors:  errors,
		client:  client,
		timeout: timeout,

		Size: prometheus.NewDesc(
			"digitalocean_volume_size_bytes",
			"Volume's size in bytes",
			labels, nil,
		),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector.
func (c *VolumeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Size
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *VolumeCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	volumes, _, err := c.client.Storage.ListVolumes(ctx, nil)
	if err != nil {
		c.errors.WithLabelValues("volume").Add(1)
		level.Warn(c.logger).Log(
			"msg", "can't list volumes",
			"err", err,
		)
		return
	}

	for _, vol := range volumes {
		labels := []string{
			strconv.Itoa(vol.DropletIDs[0]),	// This isn't actual volume ID just for relating to droplet
			vol.Name,
			vol.Region.Slug,
		}

		ch <- prometheus.MustNewConstMetric(
			c.Size,
			prometheus.GaugeValue,
			float64(vol.SizeGigaBytes*1024*1024*1024),
			labels...,
		)
	}
}
