FROM ghcr.io/oracle/oraclelinux:8-slim

COPY rpms /tmp/

RUN microdnf update && \
    microdnf install -y e2fsprogs xfsprogs && \
    microdnf clean all && \
    rpm -i /tmp/*.rpm && \
    rm /tmp/*.rpm &&  \
    rm /bin/sh /usr/bin/sh /bin/bash /usr/bin/bash &&  \
    rm /bin/cat /usr/bin/cat /bin/more /usr/bin/more

ENTRYPOINT ["/ovirt-csi-driver"]
