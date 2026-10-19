# Demonstration-only check image: the project Go runtime plus Python and the
# independent library oracle, outside every Agent's write area.
ARG BASE
FROM ${BASE}
USER root
RUN apt-get update \
 && apt-get install --yes --no-install-recommends python3 \
 && rm -rf /var/lib/apt/lists/*
COPY evaluator/ /opt/oracle/evaluator/
COPY check.py /opt/oracle/check.py
RUN chmod -R a-w /opt/oracle
USER 1000:1000
