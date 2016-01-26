FROM postgres:9.4

ENV PGPOOL_VERSION 3.4.6
ENV PGPOOL_CONF_HOME /usr/local/etc

RUN apt-get update

# pgpool port
EXPOSE 5432

ADD pgpool.conf ${PGPOOL_CONF_HOME}/pgpool.conf
ADD pool_hba.conf ${PGPOOL_CONF_HOME}/pool_hba.conf

WORKDIR /
COPY start.sh /start.sh
RUN chmod +x /start.sh

RUN mkdir -p /var/run/pgpool

CMD ["/start.sh"]
