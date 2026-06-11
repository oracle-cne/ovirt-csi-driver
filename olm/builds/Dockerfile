FROM ghcr.io/oracle/oraclelinux:9-slim

COPY rpms /tmp/

RUN microdnf update --enablerepo='ol9_UEKR7' && \
    microdnf install -y --enablerepo='ol9_UEKR7' /tmp/*.rpm
    microdnf clean all && \
    rm /tmp/*.rpm

ENTRYPOINT ["/ovirt-csi-driver"]
