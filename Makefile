GOBIN=$(GOPATH)/bin
APP_DIR_LIST=$(shell go list ./... | grep -v /vendor/)
PROJECT_VERSION=`grep VERSION: manifest.yml | cut -d ":" -f 2 | tr -d "\" "`
COMMIT_COUNT=`git rev-list --count origin/master`
COMMIT_SHA=`git rev-parse HEAD`
VERSION=$(PROJECT_VERSION)
all: build

build: bin/app
	rm -Rf application && mkdir application
	cp -Rf $(GOBIN)/tap application/kubernetes-broker

build_ms: bin/app
	rm -Rf application && mkdir application
	cp -Rf $(GOBIN)/container_broker application/
	cp -Rf $(GOBIN)/template_repository application/

bin/app: verify_gopath
	CGO_ENABLED=0 go install -tags netgo $(APP_DIR_LIST)
	go fmt $(APP_DIR_LIST)

docker_build_template_repository: build_ms
	cp -f application/template_repository app/template_repository/template_repository
	cp -Rf catalogData app/template_repository/catalogData
	docker build -t tap/template_repository app/template_repository
	rm -f app/template_repository/template_repository
	rm -Rf app/template_repository/catalogData

docker_build_container_broker: build_ms
	cp -f application/container_broker app/container_broker/container_broker
	cp -Rf catalogData app/container_broker/catalogData
	docker build -t tap/container_broker app/container_broker
	rm -f app/container_broker/container_broker
	rm -Rf app/container_broker/catalogData

docker_build_kubernetes_broker: build
	cp -f application/kubernetes-broker app/tap/kubernetes-broker
	cp -Rf catalogData app/tap/catalogData
	docker build -t tap/kubernetes_broker app/tap
	rm -f app/tap/kubernetes-broker
	rm -Rf app/tap/catalogData

local_bin/app: verify_gopath
	CGO_ENABLED=0 go install -tags local $(APP_DIR_LIST)
	go fmt $(APP_DIR_LIST)

run: local_bin/app
	./scripts/start_tap.sh

run_template_repository: local_bin/app
	./scripts/start_template_repository.sh

run_container_broker: local_bin/app
	./scripts/start_container_broker.sh

bin/govendor: verify_gopath
	go get -v -u github.com/kardianos/govendor

bin/goconvey: verify_gopath
	go get -v -u github.com/smartystreets/goconvey

bin/gomock: verify_gopath
	go get -v -u github.com/golang/mock/mockgen

deps_fetch_newest: bin/govendor
	$(GOBIN)/govendor remove +all
	@echo "Update deps used in project to their newest versions"
	$(GOBIN)/govendor fetch -v +external, +missing

deps_fetch_specific: bin/govendor
	@if [ "$(DEP_URL)" = "" ]; then\
		echo "DEP_URL not set. Run this comand as follow:";\
		echo " make deps_fetch_specific DEP_URL=github.com/nu7hatch/gouuid";\
	exit 1 ;\
	fi
	@echo "Fetchinf specific deps in newest versions"
	
	$(GOBIN)/govendor fetch -v $(DEP_URL)

deps_update: verify_gopath
	$(MAKE) bin/govendor
	@echo "Update all vendor deps according to their current version in GOPATH"
	$(GOBIN)/govendor remove +all
	$(GOBIN)/govendor update +external
	@echo "Done"

deps_list: bin/govendor
	@echo "Project dependencies list:"
	$(GOBIN)/govendor list

verify_gopath:
	@if [ -z "$(GOPATH)" ] || [ "$(GOPATH)" = "" ]; then\
		echo "GOPATH not set. You need to set GOPATH before run this command";\
		exit 1 ;\
	fi

login:
	cf login -a https://api.gotapaas.eu

logf:
	./scripts/cf-logf.sh
	
logs:
	./scripts/cf-logs.sh
	
update:
	./scripts/cf-updatesvc.sh

mock_update: bin/gomock
	$(GOBIN)/mockgen -source=app/tap/cfapi.go -package=main -destination=app/tap/cfapi_mock_test.go
	$(GOBIN)/mockgen -source=k8s/k8sfabricator.go -package=k8s -destination=k8s/k8sfabricator_mock.go
	$(GOBIN)/mockgen -source=k8s/k8screator_rest_api.go -package=k8s -destination=k8s/k8screator_rest_api_mock.go
	$(GOBIN)/mockgen -source=state/state.go -package=state -destination=state/state_mock.go
	$(GOBIN)/mockgen -source=consul/consul_service.go -package=consul -destination=consul/consul_service_mock.go

tests: verify_gopath mock_update
	go test --cover $(APP_DIR_LIST)

kate:
	kate Makefile app/* *.sh *.yml $(shell find ./catalogData/ -name '*.json')

push: build
	./scripts/cf-push.sh

pack: build
	echo "commit_sha=$(COMMIT_SHA)" > build_info.ini
	zip -r -q kubernetes-broker-${VERSION}.zip application catalogData template manifest.yml build_info.ini

pack_prepare_dirs:
	test -d "application" || mkdir application
	mkdir -p ./temp/src/github.com/trustedanalytics
	ln -sf `pwd` temp/src/github.com/trustedanalytics

pack_anywhere: pack_prepare_dirs
	$(eval GOPATH=$(shell cd ./temp; pwd))
	$(eval LOCAL_APP_DIR_LIST=$(shell cd temp/src/github.com/trustedanalytics/kubernetes-broker; GOPATH=$(GOPATH) go list ./... | grep -v /vendor/))
	GOPATH=$(GOPATH) CGO_ENABLED=0 go install -tags netgo $(LOCAL_APP_DIR_LIST)
	cp -Rf $(GOPATH)/bin/tap application/kubernetes-broker
	echo "commit_sha=$(COMMIT_SHA)" > build_info.ini
	zip -r -q kubernetes-broker-${VERSION}.zip application catalogData template manifest.yml build_info.ini
	rm -Rf ./temp

.PHONY: bin/app clean save get run update logs logf push clean
	
