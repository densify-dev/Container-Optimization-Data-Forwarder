To run the container using docker you will need to pass the config.cfg to the container. You are using a volume mount in this example:
1. Download a copy of the [config.cfg](../../densify/config/config.cfg)
2. Update the config.cfg to point to your Prometheus and Densify servers.
3. Run the container. Use the following command:
```bash
docker run -v "/config/config.cfg":"/config/config.cfg" \
  densify/container-optimization-data-forwarder
```
This command expects the config.cfg file to be located in the /config directory on the local server and this should be the same directory to which you are mounting the file within the container. 
4. Densify then loads the collected data.

The pod will run and send data over once it has done this it will end so you will need to schedule it to run based on interval you using for the data collection in the config.
