To run the container using Docker you will need to pass the config.properties to the container. You are using a volume mount in this example:
1. Download a copy of the [config.properties](../../config/config.properties) to your \<local folder\>
2. Update the config.properties to point to your Densify instance and Prometheus server.
3. Run the container using the following command:
```bash
docker run -v "<local folder>/config.properties":"/home/densify/config/config.properties":ro \
  densify/container-optimization-data-forwarder:3
```
4. Densify then loads the collected data.

The container will run and send the collected data to Densify. Once the data has been sent the container will exit. This is a one-time run, to schedule the container to collect and send data to Densify hourly, see the Kubernetes cron job option [here](../CronJob/readme.md). 
