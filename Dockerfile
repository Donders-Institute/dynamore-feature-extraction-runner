# stage 0: compile go program
FROM golang:1.20 as build
RUN mkdir -p /tmp/app
WORKDIR /tmp/app
ADD main.go .
ADD util ./util
ADD go.mod .
ADD go.sum .
RUN ls -l /tmp/app && GOOS=linux go build -a -installsuffix cgo -o bin/dfe_runnerd main.go

# stage 1: build image for the 
FROM centos:7

# application metadata
LABEL donders.ru.nl.app_name "dynamore-feature-extraction-runner"
LABEL donders.ru.nl.app_maintainer "h.lee@donders.ru.nl"
LABEL donders.ru.nl.app_code_repository "https://github.com/Donders-Institute/dynamore-feature-extraction-runner"

# required RPMs
RUN ulimit -n 1024 && yum install -y sssd-client && yum clean all && rm -rf /var/cache/yum/*

# environment variables
ENV REDIS_URL=redis://localhost:6379/0
ENV REDIS_PAYLOAD_CHANNEL=dynamore_feature_extraction
ENV FEATURE_STATS_EXEC=/opt/dynamore/run-feature-stats.sh
ENV SSH_KEY_DIR=/root/.ssh/dfe_runner
ENV QSUB_EXEC=/bin/qsub
ENV TORQUE_JOB_REQUIREMENT="walltime=1:00:00,mem=4gb"
ENV TORQUE_JOB_NAME=feature_state
ENV EXEC_USER=root
ENV PAYLOAD_OUTPUT_ROOT=
ENV TORQUE_JOB_QUEUE=

# copy binary from the build stager
WORKDIR /root
COPY --from=build /tmp/app/bin/dfe_runnerd .

## entrypoint in shell form
ENTRYPOINT ["./dfe_runnerd"]
