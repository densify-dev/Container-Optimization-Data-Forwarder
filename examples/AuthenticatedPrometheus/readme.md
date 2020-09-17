In this example you setup the Forwarder to connect to an authenticated Prometheus setup. This is meant for cases such as OpenShift default monitoring setup where the Prometheus server is setup for authentication even when using the internal kubernetes service name. If you have tried the cronjob example and received an x509 or 403 error then you likely need to use this. 

You will edit the configmap.yml, create a service account, cluster role, and cluster role binding. You will then create a pod to test the setup is working before enabling a cron job to run data collection every hour.
1. Modify the configmap.yml to point to your Densify instance and Prometheus server.
2. Create the service account:
    
    `kubectl create -f serviceaccount.yml`

3. Create the cluster role:
    
    `kubectl create -f clusterrole.yml`

4. Modify the cluster role binding to set the namespace being used to run the forwader:

	`namespace: <namespace using for Forwarder>`

5. Create the cluster role binding:
    
    `kubectl create -f clusterrolebinding.yml`

6. Create the config map:
    
    `kubectl create -f configmap.yml`
	
7. Create the pod using pod.yml: 
    
    `kubectl create -f pod.yml`
	
8. Review the log for the container you should see line similar to the following near the end of the log. If the value is 7 files then likely have issues and may need to review the rest of the log and contact Densify. If the number is higher then 7 usually between 20-70 files then can move to next step.

	`kubectl logs densify`
	
	2020-09-17T12:00:38.266347376Z 	zipped file: data
	
	2020-09-17T12:00:38.266386035Z 	uploading gke.zip; contents of 66 file(s)...
	
9. Create the cron job using the cronjob.yml 
    
    `kubectl create -f cronjob.yml`

The cron job will run and send data to Densify hourly. You should adjust the cron job schedule to run on the same interval you are using for data collection, as defined in the config.properties file.
