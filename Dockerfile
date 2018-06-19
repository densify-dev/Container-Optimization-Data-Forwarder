FROM openjdk
RUN apt-get update
RUN apt-get install -y python3
RUN apt-get install -y python-setuptools
RUN easy_install pip
RUN pip install requests
COPY ./trans .
CMD ["java", "-jar", "IngestionClient.jar", "-c", "-n", "k8s_transfer", "-l", "k8s_transfer", "-o", "upload", "-r", "-e", "-m", "k8s"]
