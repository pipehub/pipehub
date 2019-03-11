FROM golang:1.12.0-stretch

ARG TAG=HEAD

WORKDIR /pipehub
RUN git clone https://github.com/pipehub/pipehub.git /pipehub \
  && cd /pipehub \
  && git checkout $TAG \
  && mv cmd/pipehub/pipehub.sample.hcl cmd/pipehub/pipehub.hcl \
  && make generate

COPY misc/docker/build/entrypoint.sh /root/entrypoint.sh

ENTRYPOINT ["/root/entrypoint.sh"]