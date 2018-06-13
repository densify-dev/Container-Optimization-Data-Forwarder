FROM openjdk
RUN apt-get update
RUN apt-get install -y python3
RUN apt-get install -y python-setuptools
RUN easy_install pip
RUN pip install requests
COPY ./cacerts /etc/ssl/certs/java/
COPY ./trans .
CMD [“python”, “/discover.py”]
