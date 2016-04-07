GO15VENDOREXPERIMENT=1

APP=journal-2-logstash

# deps
deps: gvt_install ## install build and test dependencies
		gvt rebuild

cover_deps:
	go get github.com/pierrre/gotestcover
	go get github.com/mattn/goveralls

gvt_install:
	go get -u github.com/FiloSottile/gvt

fix_circle_go:
	scripts/install-go.sh

# tests
test: ## run unit tests
	go test -v $$(go list ./... | grep -v /vendor/)

cov: cover_deps
	gotestcover -coverprofile=coverage.out $$(go list ./... | grep -v /vendor/)

coveralls: cov ## update coveralls (requires $COVERALLS_TOKEN set)
	goveralls -repotoken $$COVERALLS_TOKEN -service=circleci -coverprofile=coverage.out

cov_html: cov ## generate coverage report in html and open a browser
	go tool cover -html=coverage.out

update_test_certs: ## make new TLS certs in the test/fixtures dir
	cd test/fixtures && ./mk-test-certs.sh

# docker demo commands
docker_up: build_linux ## start a set of docker containers to demonstrate shipping of logs from journald to logstash. See README for more details.
	docker-compose -f test/docker-compose.yml build
	docker-compose -f test/docker-compose.yml up

docker_cleanup: ## remove docker demo containers
	docker-compose -f test/docker-compose.yml down

docker_log: ## usage:  echo "hello there" | make demo_log
	@docker exec -i test_logger_1 "systemd-cat"

# build / compile
clean: ## remove transient build and test artifacts
	rm -f $(APP) coverage.out

build_linux: *.go */*.go ## build linux binary
	CGO_ENABLED=0 GOOS=linux go build -a .

build_osx: *.go */*.go ## build OSX binary
	CGO_ENABLED=0 GOOS=darwin go build -a .

# package/deploy
build_rpm: $(APP) ## build rpm
	bash scripts/build-rpm.sh

help: ## print list of tasks and descriptions
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

.PHONY: deps gvt_install clean test rpm
