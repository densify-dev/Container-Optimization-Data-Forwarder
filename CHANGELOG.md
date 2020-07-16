## 2.2.0
* Add support for node groups
* Use default logger
* Add node info metric
* Bug fix for network and disk rates on nodes

## 2.1.2

* Refactor and update to support new and older versions of Kubernetes for number of metrics. 

## 2.1.1

* Fix node memory metric name

## 2.1.0

* Added additional workloads

## 2.0.2

* Update to permissions of data directory for when running as other users.

## 2.0.1 

* Add support for passing parameters via Environment variables
* Add cluster level metrics
* Update log handling
* Fixed bug in queries where could have error grouping rows based on duplicates
* Fixed bug in queries for counter metrics that needed to be rated
* Cleaned up unused files in the project

## 2.0.0

* Converted project from Pyhton and Java to Go
* Added Node level metrics
* Added Deployment and HPA support
* Created multistage Docker build
* Added Cronjob example

## 1.0.2

* Add support for specifying the cluster to use for cases when Prometheus address names were identical across environments

## 1.0.1

* Changed to use Alpine openJDK base image.

## 1.0.0

* Initial release of container data collection from Prometheus
