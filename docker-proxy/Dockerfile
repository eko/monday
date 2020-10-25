FROM alpine:3.12

RUN apk add --no-cache openssh && \
    sed -i -e 's/Forwarding no/Forwarding yes/g' \
           -e 's/GatewayPorts no/GatewayPorts yes/g' /etc/ssh/sshd_config && \
    echo -e "Port 5022\nPermitTunnel yes\nPermitRootLogin yes\nPermitEmptyPasswords yes\nClientAliveInterval 1\nClientAliveCountMax 10\n" >> /etc/ssh/sshd_config && \
    echo -e "net.ipv6.conf.all.forwarding = 1" >> /etc/sysctl.conf && \
    passwd -d root && \
    ssh-keygen -A

EXPOSE 5022

CMD ["/usr/sbin/sshd", "-D"]
