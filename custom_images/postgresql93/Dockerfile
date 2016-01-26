FROM frodenas/postgresql:9.3
MAINTAINER Adrian Weber <adrian.weber@intel.com>

# Add scripts
ADD scripts /scripts
RUN chmod +x /scripts/*.sh

# Command to run
ENTRYPOINT ["/scripts/run.sh"]
CMD [""]
