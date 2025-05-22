FROM ghcr.io/oracle/oraclelinux:8-slim

COPY rpms /tmp/

RUN microdnf update && \
    microdnf install -y e2fsprogs xfsprogs && \
    microdnf clean all && \
    rpm -i /tmp/*.rpm && \
    rm /tmp/*.rpm &&  \
    rm -rf /bin/* /sbin/* /usr/bin/* /usr/sbin/*

ENTRYPOINT ["/ovirt-csi-driver"]
