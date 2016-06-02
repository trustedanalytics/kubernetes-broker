# Kubernetes-broker

## A better way of deploying clustered services on TAP

This repo contains an broker application, responsible for communication between CloudFoundry and Kubernetes cluster, in order to create Marketplace Services on the Kubernetes existing side-to-side with TAP.



## Building instructions

It requires go 1.6, grab it from here: https://storage.googleapis.com/golang/go1.6.linux-amd64.tar.gz

* Adjust your GOROOT to point to go 1.6
* GOPATH has to be set
* clone this repo into $GOPATH/src/github.com/trustedanalytics/kubernetes-broker
* `make run` to start locally
* `make push` to push to CF
* `make tests` to run unit tests

Please note that shell scripts are provided temporarly for convinience - they will be gone later on.

## Cloud foundry installation requirements
* Demiurge app is required to be running (https://github.com/trustedanalytics/demiurge) and its credentials has to be known.
Having this knowledge user provided service has to be added:

    ```
    cf cups kubernetes-creator-credentials -p '{"username":"demiurge_username","password":"demiurge_password","url":"demiurge_URL"}'
    ```

    e.g.:

     ```
    cf cups kubernetes-creator-credentials -p  '{"username":"admin","password":"admin","url":"http://demiurge.dev.example.com"}'
    ```
* 'sso' user provided service with cloud foundry credentials has to be created

## Modus Operandi

Upon start, broker scans it's `catalog structure` (described below), in order to be able to return CF-requested /catalog data.

After that, it listens for 5 possible API calls:

* Get CF Catalog
* Create Service
  * Asks for Kubernetes cluster details for organization;
  * Processes metadata, fills Kubernetes JSON metadata files with proper values (e.g. labels, like service_id)
  * Calls Kubernetes API and created Replication Controllers, Services and ServiceAccounts.
* Delete Service
  * Not yet implemented. Should call Kubernetes API, DELETE option for resources with label service_id = <svc id to delete>
* Create Binding
  * Asks for Kubernetes cluster details for organization;
  * Queries Kubernetes API for all the POD details for particular service_id (by label)
  * Extract environmental variables
  * Processes credentials-mappings.json - fills it with ones retrieved from Kubernetes
  * Returns CF-compatible object.
* Delete Binding
  * Not implemented; should do nothing anyway.

## Catalog structure

In broker's directory there is an folder named `catalog`. It has subdirectories per `service`.
In each `service` directory, there are two files and a directory per `service plan`:

* service.json - contains CloudFoundry required service metadata;
* credentials-mappings.json - describes `credentials` CF returns on service bindings:

  * Values prefixed with $env_<somename> are replaced with environment variables values named <somename>.
  * Values prefixed with $port_<int> are replaced with exposed container ports <int>
  * Other $-prefixed values are replaced with runtime Kubernetes data

Every `service plan` directory contains:

* plan.json - contains CloudFoundry required plan metadata; plan.json and service.json got merged when CF asks for /catalog.
* k8s/ directory, containing:
  * replicationcontroller*.json
    * one of more Kubernetes' `replication controllers` JSON schema, which can contain $-prefixed values - those will be filled by the kubernetes-broker.
  * service*.json
      * one of more Kubernetes' `service` JSON schema, which can contain $-prefixed values - those will be filled by the kubernetes-broker.*
  * account*.json
      * one of more Kubernetes' `service accounts` JSON schema, which can contain $-prefixed values - those will be filled by the kubernetes-broker.

At this point, please create new services based on the existing ones, as the schema is not stable.

Typical core labels are:

```json
"labels": {
      "org": "$org",
      "space": "$space",
      "catalog_service_id": "$catalog_service_id",
      "catalog_plan_id": "$catalog_plan_id",
      "service_id": "$service_id",
      "idx_and_short_serviceid": "$idx_and_short_serviceid",
      "managed_by": "TAP"
      }
```

Vars like $random1 to $random9 are being filled with a short random text string.

## Implemented Providers and clustered services

Most of our providers works out-of-box, but few of them requires additional configuration.
Some clustered services needs access to private or public image repository where builded images can be found.

[Images] (custom_images) can be build using simple docker command
 
 ```
 docker build -t mysql56-cluster .
 ```

One has to build provided [custom images](custom_images) and provide to broker informations how to connect to repository:


- KUBE_REPO_USER: user to authenticate
- KUBE_REPO_PASS: password for user
- KUBE_REPO_URL: url to repository where docker images were published
- KUBE_REPO_MAIL: user email

 
More info can be find on their catalogs:
* [mongodb-cluster](catalogData/mongodb-cluster/README.md)
* [mySQL-clustered](catalogData/mysql56-clustered/README.md)
* [PostgreSQL-clustered](catalogData/postgresql94-clustered/README.md)

## Log levels

You can set desired log level by setting system variable `BROKER_LOG_LEVEL_`. Available levels are:
* "CRITICAL"
* "ERROR"
* "WARNING"
* "NOTICE"
* "INFO" (default - when variable is not set)
* "DEBUG"

## Dynamic services

One can use own image to provide new service offering in catalog. For now there is no persistence for dynamic offering
(all information are in memory of broker till it restarts). More information how to add such offerings can be found
[here](catalog/README.md)

## Docker Microservices

* To build Template-repository microservice run:
    ```
    make docker_build_template_repository
    ```

    To run created docker image call:
    ```
    docker run tap/template_repository
    ```

    To run created docker image with custom envs, use:
    ```
    docker run -e BROKER_LOG_LEVEL='WARN' tap/template_repository
    ```

    Default values can be found in [Template-repository Dockerfile](app/template_repository/Dockerfile)

* To build Container-broker microservice run use same commands as above with "container_broker" param instead of "template_repository"
    [Container-broker Dockerfile](app/container_broker/Dockerfile)