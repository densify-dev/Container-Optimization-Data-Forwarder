This example shows you how to setup the Data Forwarder to connect to an authenticated Prometheus configuration. This is typically the case for OpenShift default monitoring setup, where the Prometheus server is setup for authentication, even if the internal kubernetes service name is used. If you have tried the CronJob example and received an x509 or 403 error, then you likely need to use this setup. 

To configure the Data Forwarder with an authenticated Prometheus, you need to edit the configmap.yml, create a service account, cluster role, and cluster role binding. To test the Forwarder setup, create a pod to ensure that data is sent to Densify before enabling the cron job to run data collection every hour.

1. Modify the configmap.yml to point to your Densify instance and the Prometheus server.
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
	
7. Create the pod to test the Forwarder using pod.yml:

    `kubectl create -f pod.yml`
	
8. Review the log for the container. 

	`kubectl logs densify`
	
	You should see lines similar to the following near the end of the log.
	
	> 2020-09-17T12:00:38.266347376Z 	zipped file: data
	
	> 2020-09-17T12:00:38.266386035Z 	uploading gke.zip; contents of 66 file(s)...
	
	If the content has 7 files, then you probably have issues with sending container data to Densify and need to review the rest of the log and contact Densify support. 
	If the content has more than 7 files (usually between 20-70 files), then you can move on to the next step.
	
9. Create the cron job using the cronjob.yml

    `kubectl create -f cronjob.yml`

The cron job will run and send data to Densify hourly. You should adjust the cron job schedule to run on the same interval you are using for data collection, as defined in the config.properties file.
