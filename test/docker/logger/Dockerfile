FROM fedora-systemd:22
MAINTAINER joeym@joeym.net

RUN dnf -y install systemd-journal-gateway
ADD systemd-journal-gatewayd.socket /etc/systemd/system/systemd-journal-gatewayd.socket
RUN systemctl enable systemd-journal-gatewayd.socket
RUN systemctl enable systemd-journald.socket

ADD journal-2-logstash.service /etc/systemd/system/journal-2-logstash.service
RUN systemctl enable journal-2-logstash.service
