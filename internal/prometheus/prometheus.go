package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cgTemp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cg",
		Help: "cg",
	})
)

func Init() {
	prometheus.MustRegister(cgTemp)
}
