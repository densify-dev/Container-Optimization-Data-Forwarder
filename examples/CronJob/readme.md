This example shows you how to setup the Data Forwarder to connect to a Prometheus server and send container data to Densify on an hourly basis. You need to edit the configmap.yml file, then create the config map to pass the settings to config.properties. To test the Data Forwarder setup, create a pod to ensure that data is sent to Densify before enabling the cron job to run data collection every hour.

1. Modify the configmap.yml to point to your Densify instance and to the Prometheus server.

2. Create the config map in Kubernetes.
    
    `kubectl create -f configmap.yml`
	
3. Create the pod using pod.yml.
    
    `kubectl create -f pod.yml`
	
4. Review the log for the container.
	
	`kubectl logs densify`
	
	You should see lines similar to the following, near the end of the log:
	
	> 2020-09-17T12:00:38.266347376Z 	zipped file: data
	
	> 2020-09-17T12:00:38.266386035Z 	uploading gke.zip; contents of 66 file(s)...
	
	If the content has 7 files, then you probably have issues with sending container data to Densify and need to review the rest of the log and contact Densify support. If the content has more than 7 files (usually between 20-70 files), then you can move on to the next step.
	
	Once the collected container data is sent to Densify, the pod ends.
		
5. Create the cron job using the cronjob.yml 
    
    `kubectl create -f cronjob.yml`

The cron job runs and sends the collected container data to Densify hourly.
You need to schedule the pod to run at the same interval that is configured for data collection, as defined in the config.properties file.
