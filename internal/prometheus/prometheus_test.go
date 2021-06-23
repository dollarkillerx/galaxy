package prometheus

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"math/rand"
	"net/url"
	"testing"
)

var (
	AccessCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
		},
		[]string{"method", "path"},
	)

	QueueGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_num_total",
		},
		[]string{"name"},
	)
)

func init() {
	prometheus.MustRegister(AccessCounter)
	prometheus.MustRegister(QueueGauge)
}

func TestPrometheus(t *testing.T) {
	app := gin.New()
	app.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"msg": "world"})
		purl, _ := url.Parse(ctx.Request.RequestURI)
		AccessCounter.With(prometheus.Labels{
			"method": ctx.Request.Method,
			"path":   purl.Path,
		}).Add(1)

		f := rand.Float64()
		QueueGauge.With(prometheus.Labels{"name": "queue_eddycjy"}).Set(f)
	})

	app.GET("/metrics", gin.WrapH(promhttp.Handler()))

	app.Run(":8187")
}
