package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/hybridgroup/gobot/platforms/intel-iot/edison"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
)

// Flags
var (
	listenAddress = flag.String("web.listen-address", ":9122", "Address to listen on for web interface and telemetry.")
	metricPath = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	sensorTempPin = flag.String("sensor.temperature.pin", "0", "Pin number where temperature sensor is attached.")
	sensorTempInterval = flag.Duration("sensor.temperature.interval", 5*time.Second, "Temperature sensor polling interval.")
	celsiusScale = flag.Bool("sensor.celsius-scale", true, "Whether to use Celsius scale, otherwise - Fahrenheit.")
)

const (
	namespace = "edison"
	staleInterval time.Duration = 5 * time.Minute
)

var (
	currentTemperature float64
	lastUpdated time.Time = time.Now()
)

type Exporter struct {
	up          	prometheus.Gauge
	totalScrapes	prometheus.Counter
}

func NewExporter() *Exporter {
	return &Exporter{
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "up",
			Help:      "Whether the sensor data is fresh.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrapes_total",
			Help:      "Total number of scrapes.",
		}),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up.Desc()
	ch <- e.totalScrapes.Desc()
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(ch)
	ch <- e.up
	ch <- e.totalScrapes
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	if time.Now().Sub(lastUpdated) > staleInterval {
		e.up.Set(0)
	} else {
		e.up.Set(1)
	}
	e.totalScrapes.Inc()

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(prometheus.BuildFQName(namespace, "sensor", "temperature"),
			"Current temperature.", nil, nil),
		prometheus.GaugeValue, currentTemperature,
	)
}

func main() {
	flag.Parse()

	// Initialize devices
	edisonAdaptor := edison.NewEdisonAdaptor("edison")
	tempSensor := gpio.NewGroveTemperatureSensorDriver(edisonAdaptor, "temp", *sensorTempPin, *sensorTempInterval)
	edisonAdaptor.Connect()
	tempSensor.Start()

	// Read temperature every X seconds
	gobot.On(tempSensor.Event("data"), func(data interface{}) {
		currentTemperature = data.(float64)
		lastUpdated = time.Now()
		if *celsiusScale == false {
			currentTemperature = currentTemperature * 1.8 + 32
		}
		log.Debugln(currentTemperature)
	})

	// Initialize prometheus exporter
	exporter := NewExporter()
	prometheus.MustRegister(exporter)

	log.Infof("Listening on: %s", *listenAddress)
	http.Handle(*metricPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head>
				<title>Edison exporter</title>
			</head>
			<body>
				<h1>Prometheus exporter for sensor metrics from Intel Edison</h1>
				<p><a href='` + *metricPath + `'>Metrics</a></p>
			</body>
			</html>
		`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
