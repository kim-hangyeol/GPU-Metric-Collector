#FROM registry.access.redhat.com/ubi7/ubi-minimal:latest
#FROM nvidia/cuda:10.0-base-ubuntu16.04
FROM nvidia/cuda:11.0.3-base-ubuntu18.04
#FROM ketidevit/kmctest:v2

ENV OPERATOR=/usr/local/bin/metric-collector \
    USER_UID=root \
    USER_NAME=metric-collector
ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=utility
# install operator binary
COPY _output/bin/metric-collector ${OPERATOR}

COPY bin /usr/local/bin

RUN chmod +x /usr/local/bin/user_setup

RUN  /usr/local/bin/user_setup

RUN chmod +x /usr/local/bin/entrypoint

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
