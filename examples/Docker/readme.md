To run the container using Docker you will need to pass the config.properties to the container. You are using a volume mount in this example:
1. Download a copy of the [config.properties](../../densify/config/config.properties)
2. Update the config.properties to point to your Densify instance and Prometheus server.
3. Run the container using the following command:
```bash
docker run -v "/config/config.properties":"/config/config.properties" \
  densify/container-optimization-data-forwarder
```
This command expects the config.properties file to be located in the /config directory, on the local server. This should be the same directory to which you are mounting the file, within the container. 
4. Densify then loads the collected data.

The pod will run and send the collected data. Once the data has been sent the pod will end. You will need to schedule the pod to run on the same interval you are using for data collection as defined in the config.properties file.
