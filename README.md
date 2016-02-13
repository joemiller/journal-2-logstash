journal-2-logstash
==================

[![Circle CI](https://circleci.com/gh/pantheon-systems/journal-2-logstash.svg?style=svg&circle-token=51203b2872b38718293c19a705e2b1777e95efc0)](https://circleci.com/gh/pantheon-systems/journal-2-logstash)
[![Coverage Status](https://coveralls.io/repos/github/pantheon-systems/journal-2-logstash/badge.svg?branch=master&t=Lgaqh7)](https://coveralls.io/github/pantheon-systems/journal-2-logstash?branch=master)

Securely ship JSON formatted logs from systemd's journald to logstash (ELK).

Features
--------

- Reads JSON formatted logs from systemd-journal-gatewayd (aka `s-j-gatewayd`)
  over HTTP on a unix socket. Local unix socket is used for extra security on
  the localhost.
- Ships logs to logstash server using TLS with mutual authentication of client
  and server.
- Saves the journal cursor periodically and on shutdown. Restarts from last
  log message on restarts. Reducing message loss.

Usage
=====

Config
------

### systemd-journal-gatewayd

The default configuration for s-j-gatewayd is to listen on a TCP socket.
Override to listen on a unix sock in a protected path that journal-2-logstash
will have access to:

- `/etc/systemd/system/systemd-journal-gatewayd.socket`

```
[Unit]
Description=Journal Gateway Service Socket

[Socket]
ListenStream=/run/systemd-journal-gatewayd.sock

[Install]
WantedBy=sockets.target
```

### journal-2-logstash

- `/etc/systemd/system/journal-2-logstash.service`:

```
[Unit]
Description=Journal 2 Logstash shipper

[Service]
# send this service's output to the console instead of the journal to avoid a logging loop.
StandardOutput=tty
StandardError=tty

Environment=JOURNAL2LOGSTASH_URL=logstash:4000
Environment=JOURNAL2LOGSTASH_DEBUG=true
Environment=JOURNAL2LOGSTASH_SOCKET=/run/systemd-journal-gatewayd.sock
Environment=JOURNAL2LOGSTASH_STATE_FILE=/etc/journal2logstash.state

# TLS keys and certs are not included in the container and should be mounted
# into /etc/certs at runtime.
Environment=JOURNAL2LOGSTASH_TLS_KEY=/etc/certs/logger.key
Environment=JOURNAL2LOGSTASH_TLS_CERT=/etc/certs/logger.crt
Environment=JOURNAL2LOGSTASH_TLS_CA=/etc/certs/ca.crt

ExecStart=/opt/journal-2-logstash/journal-2-logstash
Restart=on-failure
RestartSec=2s

[Install]
WantedBy=multi-user.target
```


### Logstash Receiver Config

Use the following configuration for the logstash receiver. This configuration
requires all clients to present a certificate trusted by a CA in the cacert
bundle.

Filtering rules convert journald's JSON format into Logstash JSON format.

```
input {
    tcp {
			port => "4000"
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

  # convert timestamp and message fields from journald format into logstash format

  if ("systemd_journal_json" in [tags]) {
    ### -- convert microsecond to millisecond timestamp for logstash
    ### -- extract into logstash `@timestamp` field.
    ### -- remove `__REALTIME_TIMESTAMP` field.
    ruby   { code   => "event['__REALTIME_TIMESTAMP'] = event['__REALTIME_TIMESTAMP'].to_i / 1000" }
    date   { match  => ["__REALTIME_TIMESTAMP", "UNIX_MS"] }
    mutate { remove => ["__REALTIME_TIMESTAMP"] }

    ### -- convert journald `MESSAGE` to logstash's `message` field.
    mutate { rename => { "MESSAGE" => "message" } }

    ### -- remove systemd fields you don't care about here
    mutate { remove => ["__CURSOR", "_BOOT_ID"] }
  }
}

#output {
#  stdout { codec => "rubydebug" }
#}
```

Developing
----------

Building:

- Run `make build` to build the `journal-2-logstash` binary.

Testing:

- Unit tests: `make test`
- Generate test coverage report: `make cov`
- Open test coverage report in a browser: `make cov_html`

### Docker demoing / testing

Requirements:

- docker
- docker-compose

Included in the repo is a `docker-compose.yml` in the `test/` directory that
will spinup two containers:

1. `logger` a fedora container running systemd as pid 1 and systemd-journald,
    systemd-journal-gatewayd, and the journal-2-logstash binary running.
2. `logstash` a logstash server instance configured to listen for JSON over
   TLS from the `logger` container.

Run `make docker_up` to start the containers. After the containers start (the
logstash container takes the longest) you should immediately see messages
flow into the `logstash` container and will be printed to stdout with the
`rubydebug` logstash output plugin.

```
$ make docker_up
...
logger_1   |
logger_1   | Welcome to Fedora 22 (Twenty Two)!
logger_1   |
logger_1   | Set hostname to <5ef5a7ea14ee>.
...
logger_1   | 2016/02/12 18:11:31 Could not load cursor (open /etc/journal2logstash.state: no such file or directory). Will start reading from 'last boot time'.
logger_1   | 2016/02/12 18:11:31 Error connecting to logstash: dial tcp 172.17.0.2:4000: getsockopt: connection refused
logger_1   | 2016/02/12 18:11:33 Could not load cursor (open /etc/journal2logstash.state: no such file or directory). Will start reading from 'last boot time'.
logger_1   | 2016/02/12 18:11:33 Error connecting to logstash: dial tcp 172.17.0.2:4000: getsockopt: connection refused

...

logstash_1 | {
logstash_1 |          "__MONOTONIC_TIMESTAMP" => "59420873093",
logstash_1 |                       "PRIORITY" => "6",
logstash_1 |                           "_UID" => "0",
logstash_1 |                           "_GID" => "0",
logstash_1 |                    "_MACHINE_ID" => "9d52a846ee1a4be2b2d6e563162c6aa3",
logstash_1 |                      "_HOSTNAME" => "a586fdbd6d5b",
logstash_1 |                "SYSLOG_FACILITY" => "3",
logstash_1 |                     "_TRANSPORT" => "journal",
logstash_1 |                      "CODE_FILE" => "../src/core/unit.c",
logstash_1 |                      "CODE_LINE" => "1412",
logstash_1 |                  "CODE_FUNCTION" => "unit_status_log_starting_stopping_reloading",
logstash_1 |              "SYSLOG_IDENTIFIER" => "systemd",
logstash_1 |                     "MESSAGE_ID" => "7d4958e842da4a758f6c1cdc7b36dcc5",
logstash_1 |                           "_PID" => "1",
logstash_1 |                          "_COMM" => "systemd",
logstash_1 |                           "_EXE" => "/usr/lib/systemd/systemd",
logstash_1 |                       "_CMDLINE" => "/usr/sbin/init",
logstash_1 |                 "_CAP_EFFECTIVE" => "3fffffffff",
logstash_1 |                "_SYSTEMD_CGROUP" => "/",
logstash_1 |                           "UNIT" => "systemd-journal-gatewayd.service",
logstash_1 |     "_SOURCE_REALTIME_TIMESTAMP" => "1455301048170788",
logstash_1 |                       "@version" => "1",
logstash_1 |                     "@timestamp" => "2016-02-12T18:17:28.171Z",
logstash_1 |                           "host" => "172.17.0.3",
logstash_1 |                           "tags" => [
logstash_1 |         [0] "journal_json"
logstash_1 |     ],
logstash_1 |                        "message" => "Starting Journal Gateway Service..."
logstash_1 | }
...
```

From another terminal you can generate log messages inside the `logger`
container:

```
$ echo "hello there" | make docker_log
```

Observe the log message flow from journald -> journal-2-logstash -> logstash:

```
logger_1   | 2016/02/12 18:19:15 [DEBUG] Received from journal: { "__CURSOR" : "s=aae5e906525c4c00be5e9d5026bbf40a;i=35c;b=7f0ea6b7d19f47f9a9bc928f32512ae7;m=ddc296aab;t=52b96b46b549a;x=e92c6b237a40211b", "__REALTIME_TIMESTAMP" : "1455301155574938", "__MONOTONIC_TIMESTAMP" : "59528276651", "_BOOT_ID" : "7f0ea6b7d19f47f9a9bc928f32512ae7", "PRIORITY" : "6", "_UID" : "0", "_GID" : "0", "_MACHINE_ID" : "9d52a846ee1a4be2b2d6e563162c6aa3", "_HOSTNAME" : "63c73cd1ec2e", "_TRANSPORT" : "stdout", "MESSAGE" : "hello there", "_PID" : "97", "_COMM" : "cat" } }

logstash_1 | {
logstash_1 |     "__MONOTONIC_TIMESTAMP" => "59528276651",
logstash_1 |                  "PRIORITY" => "6",
logstash_1 |                      "_UID" => "0",
logstash_1 |                      "_GID" => "0",
logstash_1 |               "_MACHINE_ID" => "9d52a846ee1a4be2b2d6e563162c6aa3",
logstash_1 |                 "_HOSTNAME" => "63c73cd1ec2e",
logstash_1 |                "_TRANSPORT" => "stdout",
logstash_1 |                      "_PID" => "97",
logstash_1 |                     "_COMM" => "cat",
logstash_1 |                  "@version" => "1",
logstash_1 |                "@timestamp" => "2016-02-12T18:19:15.574Z",
logstash_1 |                      "host" => "172.17.0.3",
logstash_1 |                      "tags" => [
logstash_1 |         [0] "journal_json"
logstash_1 |     ],
logstash_1 |                   "message" => "hello there"
logstash_1 | }

```

TODO
----

- Make TLS client cert optional. This will be Useful for connecting to
  logz.io's TCP/TLS receiver which doesn't use client certs.  workaround:
  specify an arbitrary client key and cert and logz.io will ignore it.
- also check TODOs in source files

Misc Notes
----------

- curl testing of `s-j-gatewayd`:

`curl -H'Accept: application/event-stream' -H 'Range: entries=:-1:1' 'localhost:19531/entries'`
