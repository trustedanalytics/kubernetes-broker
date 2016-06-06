GOBIN=$(GOPATH)/bin
APP_DIR_LIST=$(shell go list ./... | grep -v /vendor/)
PROJECT_VERSION=`grep VERSION: manifest.yml | cut -d ":" -f 2 | tr -d "\" "`
COMMIT_COUNT=`git rev-list --count origin/master`
COMMIT_SHA=`git rev-parse HEAD`
VERSION=$(PROJECT_VERSION)
all: build

build: bin/app
	@echo "build complete."

bin/app: verify_gopath
	CGO_ENABLED=0 go install -tags netgo $(APP_DIR_LIST)
	go fmt $(APP_DIR_LIST)

local_bin/app: verify_gopath
	CGO_ENABLED=0 go install -tags local $(APP_DIR_LIST)
	go fmt $(APP_DIR_LIST)

run: local_bin/app
	./scripts/start.sh

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

push: build
	test -d "application" || mkdir application
	cp -Rf $(GOBIN)/kubernetes-broker application
	./scripts/cf-push.sh

logf:
	./scripts/cf-logf.sh
	
logs:
	./scripts/cf-logs.sh
	
update:
	./scripts/cf-updatesvc.sh

mock_update: bin/gomock
	$(GOBIN)/mockgen -source=cfapi.go -package=main -destination=cfapi_mock_test.go
	$(GOBIN)/mockgen -source=k8s/k8sfabricator.go -package=k8s -destination=k8s/k8sfabricator_mock.go
	$(GOBIN)/mockgen -source=k8s/k8screator_rest_api.go -package=k8s -destination=k8s/k8screator_rest_api_mock.go
	$(GOBIN)/mockgen -source=state/state.go -package=state -destination=state/state_mock.go
	$(GOBIN)/mockgen -source=consul/consul_service.go -package=consul -destination=consul/consul_service_mock.go

tests: verify_gopath mock_update
	go test --cover $(APP_DIR_LIST)

kate:
	kate Makefile app/* *.sh *.yml $(shell find ./catalogData/ -name '*.json')

pack: build
	test -d "application" || mkdir application
	cp -Rf $(GOBIN)/kubernetes-broker application
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
	cp -Rf $(GOPATH)/bin/kubernetes-broker application
	echo "commit_sha=$(COMMIT_SHA)" > build_info.ini
	zip -r -q kubernetes-broker-${VERSION}.zip application catalogData template manifest.yml build_info.ini
	rm -Rf ./temp

.PHONY: bin/app clean save get run update logs logf push clean
	
