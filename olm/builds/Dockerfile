FROM ghcr.io/oracle/oraclelinux:9-slim

COPY rpms /tmp/

RUN microdnf update && \
    microdnf install -y e2fsprogs xfsprogs && \
    microdnf clean all && \
    rpm -i /tmp/*.rpm && \
    rm /tmp/*.rpm

ENTRYPOINT ["/ovirt-csi-driver"]
