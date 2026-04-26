FROM alpine:3.20

RUN apk add --no-cache openssh-server \
    && adduser -D -s /bin/sh deploy \
    && echo "deploy:docker-test" | chpasswd \
    && mkdir -p /run/sshd /srv/site/public /srv/site/content \
    && printf "%s\n" "PasswordAuthentication no" "PubkeyAuthentication yes" >> /etc/ssh/sshd_config \
    && ssh-keygen -A \
    && chown -R deploy:deploy /srv/site

COPY entrypoint.sh /entrypoint.sh

EXPOSE 22

ENTRYPOINT ["/entrypoint.sh"]
