FROM ghcr.io/oracle/oraclelinux:9-slim

COPY rpms /tmp/

RUN microdnf update --enablerepo='ol9_UEKR7' && \
    microdnf install --enablerepo='ol9_UEKR7' -y e2fsprogs xfsprogs && \
    microdnf clean all && \
    rpm -i /tmp/*.rpm && \
    rm /tmp/*.rpm

ENTRYPOINT ["/ovirt-csi-driver"]
