FROM alpine:3.10

LABEL maintainer="Siman Chou <https://github.com/simanchou/dnswm>"

RUN echo "http://mirrors.ustc.edu.cn/alpine/v3.10/main" >/etc/apk/repositories && echo "http://mirrors.ustc.edu.cn/alpine/v3.10/community" >>/etc/apk/repositories
RUN apk add --update --no-cache  --allow-untrusted supervisor

RUN mkdir /dnswm
WORKDIR /dnswm
COPY docker/start.sh .
COPY coredns .
COPY dnswm .
COPY assets ./assets
COPY tmpl ./tmpl

EXPOSE 53
EXPOSE 9001

CMD ["./start.sh"]
