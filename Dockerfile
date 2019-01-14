FROM openjdk
RUN apt-get update
RUN apt-get install -y python3
RUN apt-get install -y python-setuptools
RUN easy_install pip
RUN pip install requests
COPY ./trans .
CMD ["java", "-jar", "IngestionClient.jar", "-c", "-n", "k8s_transfer_v2", "-l", "k8s_transfer_v2", "-o", "upload", "-r", "-S", "config"]
