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
