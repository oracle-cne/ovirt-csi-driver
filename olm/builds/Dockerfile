FROM ghcr.io/oracle/oraclelinux:9-slim

RUN microdnf update --enablerepo='ol9_UEKR7' && \
    microdnf install -y --enablerepo='ol9_UEKR7' ovirt-csi-driver && \
    microdnf clean all

ENTRYPOINT ["/ovirt-csi-driver"]
