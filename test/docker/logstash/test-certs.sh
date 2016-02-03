#!/bin/sh
set -e

# reset/cleanup
rm -rf certs
mkdir certs

# create CA
echo "==> Creating new CA: certs/ca.key, certs/ca.crt"
openssl req -new -newkey rsa:2048 -days 3650 -nodes -x509 \
	-subj '/CN=test-CA/O=testco/C=US' \
  -keyout certs/ca.key \
  -out certs/ca.crt

cat certs/ca.key certs/ca.crt >certs/ca.pem

# create key/cert for logstash
echo "==> Creating key/cert for 'logstash': certs/logstash.key, certs/logstash.crt"
openssl req -new -newkey rsa:2048 -subj '/CN=logstash' -nodes \
	-keyout certs/logstash.key \
  -out certs/logstash.csr
openssl x509 -req -days 365 -set_serial 02 \
  -CA certs/ca.crt \
  -CAkey certs/ca.key \
  -extfile openssl-test.cfg \
  -extensions v3_ca \
  -in certs/logstash.csr \
  -out certs/logstash.crt
cat certs/logstash.key certs/logstash.crt >certs/logstash.pem

# create java keystore for logstash key and cert
echo "==> Creating PKCS12: certs/logstash.p12"
openssl pkcs12 -chain -export \
	-CAfile certs/ca.crt \
  -password pass:password \
	-inkey certs/logstash.key \
	-in certs/logstash.crt \
	-name logstash \
	-out certs/logstash.p12
echo "==> Creating Java Keystore: certs/logstash.ks"
keytool -importkeystore \
	-srckeystore certs/logstash.p12 \
	-srcstoretype pkcs12 \
	-srcstorepass password \
	-destkeystore certs/logstash.ks \
	-deststoretype JKS \
	-storepass password
keytool -importcert \
	-alias ca \
	-noprompt \
	-trustcacerts \
	-file certs/ca.crt \
	-keystore certs/logstash.ks \
	-storepass password

ls -l $(find ./certs -type f)
