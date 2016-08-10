## 0.4.1 (2016-08-10)

* Move the updating of lag_seconds metric into a go-routine so that it can be updated
  when the main loop is hung, improving visibility of instances that may become
  hung.

## 0.4.0 (2016-08-05)

* Added write timeout option `-o`. Default is 10 seconds. This should help with occasional
  issues observed in previous versions where journal-2-logstash would seemingly hang and
  cease sending logs to Logstash. This was observable by increasing lag_seconds metrics
  and the cessation of the period reconnects to the Logstash server.

## 0.3.1 (2016-05-23)

* Handle cases when the journal will encode field values as byte arrays for fields other than the `MESSAGE` field.

## 0.3.0 (2016-05-22)

* Added periodic reconnect to Logstash server to improve load-balancing in environments with multiple Logstash
  servers.
* Logging changes to improve inspecting the state of a running journal-2-logstash instance.
* Send-metrics-to-graphite now supported.
* Added `seconds_behind` metric. This allows for inspecting a running instance of journal-2-logstash and
  determining the progress (or lack of) in the log stream.

## 0.2.0 (2016-04-17)

* JSON messages from systemd-journal-gatewayd are now parsed and converted into Logstash V1 event format before
  being sent to the Logstash server.

## 0.1.0 (2016-02-12)

* initial release
