# A clean Debian 12 system with systemd, for the install test only.
FROM debian:12@sha256:d74df7598108a173a621eda4d58a2adbf850b2d9a9d81bc2cca1a6c21174a880
RUN apt-get update \
	&& apt-get install -y --no-install-recommends systemd systemd-sysv dbus curl ca-certificates busybox iproute2 \
	&& rm -rf /var/lib/apt/lists/* \
	&& systemctl mask getty.target console-getty.service systemd-firstboot.service systemd-logind.service
STOPSIGNAL SIGRTMIN+3
CMD ["/lib/systemd/systemd"]
