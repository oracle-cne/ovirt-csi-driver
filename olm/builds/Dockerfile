FROM ghcr.io/oracle/oraclelinux:9-slim

COPY rpms /tmp/

RUN microdnf update --enablerepo='ol9_UEKR7' && \
    microdnf install -y --enablerepo='ol9_UEKR7' systemd-udev e2fsprogs xfsprogs util-linux-core util-linux && \
    microdnf clean all && \
    rpm -Uvh /tmp/*.rpm && \
    rm /tmp/*.rpm

ENTRYPOINT ["/ovirt-csi-driver"]
