package main

import (
	"flag"
	"math"
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
	listenAddress         = flag.String("web.listen-address", ":9122", "Address to listen on for web interface and telemetry.")
	metricPath            = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	sensorTempPin         = flag.String("sensor.temperature.pin", "0", "Analog pin number where temperature sensor is attached.")
	sensorLightPin        = flag.String("sensor.light.pin", "1", "Analog pin number where light sensor is attached.")
	sensorSoundPin        = flag.String("sensor.sound.pin", "3", "Analog pin number where sound sensor is attached.")
	sensorPollingInterval = flag.Duration("sensor.polling.interval", 5*time.Second, "Sensor polling interval.")
)

const (
	namespace                   = "edison"
	staleInterval time.Duration = 5 * time.Minute
)

var (
	celsius, fahrenheit, sound, light       float64
	tempUpdated, soundUpdated, lightUpdated time.Time
)

type Exporter struct {
	up           prometheus.Gauge
	totalScrapes prometheus.Counter
}

func NewExporter() *Exporter {
	return &Exporter{
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "up",
			Help:      "Whether exporter is up.",
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
	e.up.Set(1)
	e.totalScrapes.Inc()

	if time.Now().Sub(tempUpdated) < staleInterval {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(prometheus.BuildFQName(namespace, "sensor", "temperature"),
				"Current temperature.", []string{"metric"}, nil),
			prometheus.GaugeValue, celsius, "celsius",
		)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(prometheus.BuildFQName(namespace, "sensor", "temperature"),
				"Temperature in C and F.", []string{"metric"}, nil),
			prometheus.GaugeValue, fahrenheit, "fahrenheit",
		)
	}
	if time.Now().Sub(soundUpdated) < staleInterval {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(prometheus.BuildFQName(namespace, "sensor", "sound"),
				"Sound (noise) level.", nil, nil),
			prometheus.GaugeValue, float64(sound),
		)
	}
	if time.Now().Sub(lightUpdated) < staleInterval {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(prometheus.BuildFQName(namespace, "sensor", "light"),
				"Luminous flux per unit area.", nil, nil),
			prometheus.GaugeValue, float64(light),
		)
	}
}

func main() {
	flag.Parse()

	// Initialize Intel Edison
	edisonAdaptor := edison.NewEdisonAdaptor("edison")
	edisonAdaptor.Connect()

	lightSensor := gpio.NewGroveLightSensorDriver(edisonAdaptor, "light", *sensorLightPin, *sensorPollingInterval)
	lightSensor.Start()
	gobot.On(lightSensor.Event("data"), func(data interface{}) {
		raw := float64(data.(int))
		// convert to lux
		resistance := (1023.0 - raw) * 10.0 / raw * 15.0
		light = 10000.0 / math.Pow(resistance, 4.0/3.0)
		lightUpdated = time.Now()
		log.Debugln("illuminance: ", light)
	})

	soundSensor := gpio.NewGroveSoundSensorDriver(edisonAdaptor, "sound", *sensorSoundPin, *sensorPollingInterval)
	soundSensor.Start()
	gobot.On(soundSensor.Event("data"), func(data interface{}) {
		sound = float64(data.(int))
		soundUpdated = time.Now()
		log.Debugln("sound level: ", sound)
	})

	tempSensor := gpio.NewGroveTemperatureSensorDriver(edisonAdaptor, "temp", *sensorTempPin, *sensorPollingInterval)
	tempSensor.Start()
	gobot.On(tempSensor.Event("data"), func(data interface{}) {
		celsius = data.(float64)
		fahrenheit = celsius*1.8 + 32
		tempUpdated = time.Now()
		log.Debugln("temperature: ", celsius)
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
				<title>IoT Edison exporter</title>
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
