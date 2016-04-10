#!/bin/sh

if [ -n "$CIRCLECI" ]; then
        sudo apt-get install rpm
fi

gem install fpm --no-rdoc --no-ri

name='journal-2-logstash'
version=$(cat VERSION.txt)
iteration="$(date +%Y%m%d%H%M).git$(git rev-parse --short HEAD)"  # datecode + git sha-ref: "201503020102.gitef8e0fb"
arch='x86_64'
url="https://github.com/pantheon-systems/${name}"
vendor='Pantheon'
description='securely ship journald logs to logstash'
install_prefix="/opt/${name}"

fpm -s dir -t rpm \
    --name "${name}" \
    --version "${version}" \
    --iteration "${iteration}" \
    --architecture "${arch}" \
    --url "${url}" \
    --vendor "${vendor}" \
    --description "${description}" \
    --prefix "$install_prefix" \
    README.md \
    VERSION.txt \
    $name

if [ -d "$CIRCLE_ARTIFACTS" ] ; then
  cp ./*.rpm "$CIRCLE_ARTIFACTS"
fi
