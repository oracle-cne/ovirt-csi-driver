FROM ghcr.io/oracle/oraclelinux:9-slim

RUN microdnf update --enablerepo='ol9_UEKR7' && \
    microdnf install -y --enablerepo='ol9_UEKR7' ovirt-csi-driver && \
    microdnf clean all && \
    rm /bin/col \
       /bin/colcrt \
       /bin/colrm \
       /bin/column \
       /bin/look \
       /bin/gencat
       /bin/hexdump \
       /bin/more \
       /bin/rename \
       /bin/script \
       /bin/scriptlive \
       /bin/scriptreplay \
       /bin/ul

ENTRYPOINT ["/ovirt-csi-driver"]
