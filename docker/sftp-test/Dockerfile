FROM alpine:3.20

RUN apk add --no-cache openssh-server openssh-sftp-server

COPY entrypoint.sh /usr/local/bin/styxpress-sftp-entrypoint
RUN chmod +x /usr/local/bin/styxpress-sftp-entrypoint

EXPOSE 22

ENTRYPOINT ["/usr/local/bin/styxpress-sftp-entrypoint"]
