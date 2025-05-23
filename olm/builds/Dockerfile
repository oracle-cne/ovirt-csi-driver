FROM ghcr.io/oracle/oraclelinux:8-slim

COPY rpms /tmp/

RUN microdnf update && \
    microdnf install -y e2fsprogs xfsprogs && \
    microdnf clean all && \
    rpm -i /tmp/*.rpm && \
    rm /tmp/*.rpm &&  \
    rm -f /bin/sh && \
    rm -f /usr/bin/sh && \
    rm -f /bin/bash && \
    rm -f /usr/bin/bash && \
    rm -f /bin/cat && \
    rm -f /usr/bin/cat && \
    rm -f /bin/more && \
    rm -f /usr/bin/more

ENTRYPOINT ["/ovirt-csi-driver"]
