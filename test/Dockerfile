FROM ubuntu:focal

# Get latest updates and install deps

RUN apt-get update
RUN apt-get dist-upgrade -y
RUN apt-get install -y wget gnupg

# Install perforce

RUN wget -qO - https://package.perforce.com/perforce.pubkey | apt-key add -
RUN echo 'deb http://package.perforce.com/apt/ubuntu focal release' > /etc/apt/sources.list.d/perforce.list
RUN apt-get update
RUN apt-get install -y helix-p4d

# Configure perforce (we're forced to make a "secure" password here)

ARG SERVER_ID
ARG CASE_INSENSITIVE
RUN /opt/perforce/sbin/configure-helix-p4d.sh $SERVER_ID -n -p 1666 -r /p4 -u super -P super123 --case $CASE_INSENSITIVE
COPY main.conf /etc/perforce/p4dctl.conf.d/main.conf

# Disable the security policy and clear the super's password (these are just tests, security is irrelevant)

RUN p4dctl start main && \
  echo "super123\nsuper123\n" | p4 -u super login && \
  p4 -u super configure set security=0 && \
  echo "super123\n\n\n" | p4 -u super passwd && \
  p4dctl stop main

# Start perforce via p4dctl, then spin forever piping the logs to stdout

EXPOSE 1666
CMD p4dctl start main && tail -F /p4/logs/log
