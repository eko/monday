FROM alpine:3.24.1

# SSH server used as a proxy inside the cluster by the "kubernetes-remote"
# forward type: Monday port-forwards its SSH port locally and opens a remote
# forward through it, so the traffic reaching the pod is sent to the local
# application.
#
# sshd keeps the first value it reads for a directive, so ours are prepended
# to the configuration file: they win over the hardened defaults shipped by
# the openssh package (AllowTcpForwarding no, GatewayPorts no...).
RUN apk add --no-cache openssh && \
    { \
      printf '%s\n' \
        "Port 5022" \
        "AllowTcpForwarding yes" \
        "GatewayPorts yes" \
        "PermitTunnel yes" \
        "PermitRootLogin yes" \
        "PermitEmptyPasswords yes" \
        "ClientAliveInterval 1" \
        "ClientAliveCountMax 10" \
        ""; \
      cat /etc/ssh/sshd_config; \
    } > /etc/ssh/sshd_config.monday && \
    mv /etc/ssh/sshd_config.monday /etc/ssh/sshd_config && \
    passwd -d root && \
    ssh-keygen -A

EXPOSE 5022

CMD ["/usr/sbin/sshd", "-D", "-e"]
