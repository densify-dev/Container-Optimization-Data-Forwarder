In this example you will edit the configmap.yml, create a pod to test then a cron job to run data collection every hour and pass the Config Map containing the required settings, to the config.properties file.
1. Modify the configmap.yml to point to your Densify instance and Prometheus server.
2. Create the config map in Kubernetes.
    
    `kubectl create -f configmap.yml`
	
3. Create the pod using pod.yml: 
    
    `kubectl create -f pod.yml`
	
4. Review the log for the container you should see line similar to the following near the end of the log. If the value is 7 files then likely have issues and may need to review the rest of the log and contact Densify. If the number is higher then 7 usually between 20-70 files then can move to next step.
	
	`kubectl logs densify`
	
	2020-09-17T12:00:38.266347376Z 	zipped file: data
	
	2020-09-17T12:00:38.266386035Z 	uploading gke.zip; contents of 66 file(s)...
	
5. Create the cron job using the cronjob.yml 
    
    `kubectl create -f cronjob.yml`

The cron job will run and send data to Densify hourly. You should adjust the cron job schedule to run on the same interval you are using for data collection, as defined in the config.properties file.
