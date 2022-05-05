This example shows you how to setup the Data Forwarder to connect to a Prometheus server and send container data to Densify on an hourly basis. You need to edit the configmap.yml file, then create the config map to pass the settings to config.properties. To test the Data Forwarder setup, create a pod to ensure that data is sent to Densify before enabling the cron job to run data collection every hour.

1. Modify the configmap.yml to point to your Densify instance and to the Prometheus server.

2. Create the config map in Kubernetes.
    
    `kubectl create -f configmap.yml`
	
3. Create the pod using pod.yml.
    
    `kubectl create -f pod.yml`
	
4. Review the log for the container.
	
	`kubectl logs densify`
	
	You should see lines similar to the following, near the end of the log:
	
	> {"level":"info","pkg":"default","time":1651699421230540,"caller":"src/densify.com/forwarderv2/files.go:88","goid":1,"message":"zipping gke_cluster.zip, contents: cluster - 21 files; container - 16 files; node - 17 files; node_group - 22 files; hpa - 4 files; rq - 7 files; crq - 0 files; total - 87 files"}
	
	> {"level":"info","pkg":"default","file":"data/gke_cluster.zip","time":1651699421321196,"caller":"src/densify.com/forwarderv2/main.go:47","goid":1,"message":"file uploaded successfully"}
	
	The exact number of files - under each subfolder and total - depends on the Data Forwarder `include_list` configuration, kube-state-metrics configuration and what is defined/running in the Kubernetes cluster we collect data for. If we use the default `include_list` configuration (empty value means collect all), we should see non-zero number of files at least for cluster, container, node and hpa. The other are cluster-specific.
	If the numbers are lower than expected, you probably have issues with sending container data to Densify and need to review the rest of the log and contact Densify support. Otherwise, you can move on to the next step.
	
	Once the collected container data is sent to Densify, the pod ends.
		
5. Create the cron job using the cronjob.yml 
    
    `kubectl create -f cronjob.yml`

The cron job runs and sends the collected container data to Densify hourly.
You need to schedule the pod to run at the same interval that is configured for data collection, as defined in the config.properties file.
