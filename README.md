journal-2-logstash
==================

TODO: coveralls badge

- optimized for speed on client. no json parsing is done.
- minimal client size
- saves the "cursor" periodically to a state file and loads at startup to improve log delivery
- TLS with mutual authentication
- Assumes systemd-journal-gateway is configured to listen on a unix socket rather than tcp

Logstash Config Example
-----------------------

Input: Configure a TCP listener using TLS authentication and the json codec.

Filtering: Perform some minimal filtering on the incoming journal json format:

1. Systemd-journald encodes the event timestamp in the `__REALTIME_TIMESTAMP` key as microseconds since epoch. Because Logstash uses the Joda time library which is limited to milliseconds precision we truncate the timestamp to milliseconds.
2. Use the `date` filter to parse the `__REALTIME_TIMESTAMP` field into the event's canonical `@timestamp` field.
3. Rename the `MESSAGE` key to `message`.
4. Drop some extraneous fields that we likely will not need.

```
input {
    tcp {
			port => "4001"
			codec => "json"
			ssl_enable => "true"
			ssl_key => "/path/to/server.pem"
			ssl_cert => "/path/to/server.pem"
			ssl_cacert => "/path/to/ca_bundle.pem"
			ssl_verify => "true"
			tags => ["systemd_journal_json"]
		 }
}

filter {
  if ("systemd_journal_json" in [tags]) {
    ruby { code => "event['__REALTIME_TIMESTAMP'] = event['__REALTIME_TIMESTAMP'].to_i / 1000" }
    date { match => ["__REALTIME_TIMESTAMP", "UNIX_MS"] }
    mutate { rename => { "MESSAGE" => "message" } }
    mutate { remove => ["__CURSOR", "__REALTIME_TIMESTAMP", "_BOOT_ID"] }
  }
}

output {
  stdout { codec => "rubydebug" }
}
```

TODO
----

- make TLS client cert optional. Useful for connecting to logz.io's TCP/TLS receiver which doesn't use client certs.
  workaround: should be able to specify an arbitrary client key and cert and logz.io will ignore it.
- also check TODOs in source files

Misc Notes
----------

curl -H'Accept: application/event-stream' -H 'Range: entries=:-1:1' 'localhost:19531/entries'



docker-compose up
- start logstash-container w/ simple tls input + rubydebug output
- start fedora-22-systemd-init with systemd-j-gw running and listening on a unix sock, and j-2-ls mounted and running in the container

echo "fake log message" | docker exec logstash-container -i "systemd-cat"


