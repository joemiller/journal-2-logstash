#!/bin/sh
set -e

# reset/cleanup
function cleanup {
	rm -rf certs
	mkdir certs
}

# create CA
function create_ca {
	echo "==> Creating new CA: certs/ca.key, certs/ca.crt"
	openssl req -new -newkey rsa:2048 -days 4650 -nodes -x509 \
		-subj '/CN=test-CA/O=testco/C=US' \
		-keyout certs/ca.key \
		-out certs/ca.crt
	cat certs/ca.key \
			certs/ca.crt >certs/ca.pem
}


# create a client key/cert signed by the CA.
# The first argument to this function is used for both the filenames and CN
#
# Example:
#    key_and_cert foo
# Results in:
#    - certs/foo.key, certs/foo.crt, certs/foo.pem
#    - Cert subject: /CN=foo/
#
function key_and_cert {
  name=$1
	if [ -z "$name" ]; then
		echo "key_and_cert missing argument"
		return 1
	fi
	echo "==> Creating key/cert for '$name': certs/$name.key, certs/$name.crt"
	openssl req -new -newkey rsa:2048 -subj "/CN=$name" -nodes \
		-keyout certs/$name.key \
		-out certs/$name.csr
	openssl x509 -req -days 3650 -set_serial 02 \
		-CA certs/ca.crt \
		-CAkey certs/ca.key \
		-extfile openssl-test.cfg \
		-extensions v3_ca \
		-in certs/$name.csr \
		-out certs/$name.crt
	cat certs/$name.key \
			certs/$name.crt >certs/$name.pem
}

cleanup
create_ca
key_and_cert "logger"
key_and_cert "logstash"

ls -l $(find ./certs -type f)
