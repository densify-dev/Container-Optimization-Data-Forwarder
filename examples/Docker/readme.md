To run the container using docker you will need to pass the config.cfg to the container we are using a volume mount in this example:
1. download a copy of the [config.cfg](../../densify/config/config.cfg)
2. Update the config.cfg to point to your Prometheus and Densify servers.
3. Run the container such as:
```bash
docker run -v "/config/config.cfg":"/config/config.cfg" \
  densify/container-optimization-data-forwarder
```
This would expect the config.cfg to be in the /config directory on the local server and that would be the same directory we are mounting it into in the container. 
4. Densify then loads the collected data.