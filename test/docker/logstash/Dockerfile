# https://hub.docker.com/_/logstash/
FROM logstash:1.5

ADD logstash.conf       /usr/local/logstash/config/logstash.conf

EXPOSE 4000

CMD logstash -f /usr/local/logstash/config/logstash.conf --debug
