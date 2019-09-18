FROM debian:stretch

LABEL maintainer="Siman Chou <https://github.com/simanchou/dnswm>"

ADD docker/sources.list /etc/apt/
RUN apt-get clean && apt-get update
RUN apt-get -y install supervisor

RUN mkdir /opt/dnswm
WORKDIR /opt/dnswm
COPY docker/start.sh .
COPY coredns .
COPY dnswm .
COPY assets ./assets
COPY tmpl ./tmpl

EXPOSE 53
EXPOSE 9001

CMD ["./start.sh"]
