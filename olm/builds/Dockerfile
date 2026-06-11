FROM ghcr.io/oracle/oraclelinux:9-slim

COPY rpms /tmp/

RUN microdnf update --enablerepo='ol9_UEKR7' && \
    rpm -i /tmp/*.rpm && \
    microdnf clean all && \
    rm /tmp/*.rpm

ENTRYPOINT ["/ovirt-csi-driver"]
