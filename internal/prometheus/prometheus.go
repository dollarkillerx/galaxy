package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cpuTemp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_temperature_celsius",
		Help: "Current temperature of the CPU. ",
	})
	cgTemp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cg",
		Help: "cg",
	})
)

func Init() {
	prometheus.MustRegister(cpuTemp)
	prometheus.MustRegister(cgTemp)
}
