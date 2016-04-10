# https://hub.docker.com/_/logstash/
FROM logstash:1.5

RUN /opt/logstash/bin/plugin install logstash-filter-translate
ADD logstash.conf       /usr/local/logstash/config/logstash.conf

EXPOSE 4000

#CMD logstash -f /usr/local/logstash/config/logstash.conf --debug
CMD logstash -f /usr/local/logstash/config/logstash.conf
