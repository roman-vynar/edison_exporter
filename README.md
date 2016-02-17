# IoT Edison exporter

[Prometheus](http://prometheus.io) exporter for sensor metrics from [Intel Edison](https://software.intel.com/en-us/iot/hardware/edison).  

Supported sensors:
 * Grove temperature sensor
 * Grove sound sensor
 * Grove light sensor

### Build and run

Build on a machine using Go 1.5+ with GOPATH set:

    git clone https://github.com/roman-vynar/edison_exporter.git
    cd edison_exporter
    go get -d
    GOARCH=386 GOOS=linux go build edison_exporter.go

Also you can download the binary from [Releases](https://github.com/roman-vynar/edison_exporter/releases)

Copy to Edison:
    
    scp edison_exporter root@<Edison IP>:~/
    
Run:

    root@edison:~# ./edison_exporter

### Options

Name                              | Description
----------------------------------|------------------------------------------------------------------------------------
-sensor.temperature.pin           |	Analog pin number where temperature sensor is attached.  (default "0")
-sensor.light.pin                 |	Analog pin number where light sensor is attached. (default "1")
-sensor.sound.pin     	          | Analog pin number where sound sensor is attached. (default "3")
-sensor.polling.interval          |	Sensor polling interval. (default 5s)
-log.level value                  | Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal, panic]. (default info)
-web.listen-address               | Address to listen on for web interface and telemetry. (default ":9122")
-web.telemetry-path               | Path under which to expose metrics. (default "/metrics")

### Samples

Output:

    $ curl -s http://<Edison IP>:9122/metrics|grep edison
    # HELP edison_exporter_scrapes_total Total number of scrapes.
    # TYPE edison_exporter_scrapes_total counter
    edison_exporter_scrapes_total 3
    # HELP edison_exporter_up Whether exporter is up.
    # TYPE edison_exporter_up gauge
    edison_exporter_up 1
    # HELP edison_sensor_light Luminous flux per unit area.
    # TYPE edison_sensor_light gauge
    edison_sensor_light 0.01644942660972975
    # HELP edison_sensor_sound Sound (noise) level.
    # TYPE edison_sensor_sound gauge
    edison_sensor_sound 1
    # HELP edison_sensor_temperature Temperature in C and F.
    # TYPE edison_sensor_temperature gauge
    edison_sensor_temperature{metric="celsius"} 19.093447311567388
    edison_sensor_temperature{metric="fahrenheit"} 66.3682051608213


Graph with Grafana:

<img src="temperature.png">

