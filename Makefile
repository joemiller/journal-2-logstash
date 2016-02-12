GO15VENDOREXPERIMENT=1

APP=journal-2-logstash

all: deps test $(APP)

list:
	@make -rqp | awk -F':' '/^[a-zA-Z0-9][^$$#\/\t=]*:([^=]|$$)/ {split($$1,A,/ /);for(i in A)print A[i]}' | sort | uniq

# deps
deps: gvt_install
		gvt rebuild

gvt_install:
		go get -u github.com/FiloSottile/gvt

# tests
test:
	go test -v $$(go list ./... | grep -v /vendor/)

cov:
	go get github.com/pierrre/gotestcover
	go get github.com/mattn/goveralls
	gotestcover -coverprofile=coverage.out $$(go list ./... | grep -v /vendor/)

coveralls: cov
	goveralls -repotoken $$COVERALLS_TOKEN -service=circleci -coverprofile=coverage.out

cov_html: cov
	go tool cover -html=coverage.out

update_test_certs:
	cd test/fixtures && ./mk-test-certs.sh

# docker demo commands
docker_up: $(APP)
	docker-compose -f test/docker-compose.yml build
	docker-compose -f test/docker-compose.yml up

docker_cleanup:
	docker-compose -f test/docker-compose.yml down

# usage:  echo "hello there" | make demo_log
docker_log:
	@docker exec -i test_logger_1 "systemd-cat"

# build / compile
clean:
	rm -f $(APP) coverage.out

$(APP): *.go */*.go
#	CGO_ENABLED=0 go build -a .
	CGO_ENABLED=0 GOOS=linux go build -a .

# package/deploy
rpm: $(APP)
	bash deploy/build-rpm.sh

.PHONY: deps gvt_install clean test rpm
