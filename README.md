# Dynamore feature extraction runner

A small deamon that submits HPC job to run feature extraction upon receiving payload message from a Redis server.

## Installation

On a Linux server running CentOS/RedHat, download the RPM package from the release assets, and run

```bash
$ yum localinstall dynamore-feature-extraction-runner-{version}-1.el7.x86_64.rpm
```

A systemd service called `dfe_runnerd` will be enabled and started.

## Configuration

Uncomment and change the variables in the file is located in `/etc/sysconfig/dfe_runnerd`.  An example can be found [here](scripts/dfe_runnerd.env).

Restart the daemon after changing the values:

```bash
$ systemctl restart dfe_runnerd
```

## Build from source

It requires [Go](https://golang.org) to compiler the source code.

```bash
$ git clone https://github.com/Donders-Institute/dynamore-feature-extraction-runner.git
$ make build
```

The executable named `dynamore-feature-extraction-runner.linux_amd64` is build into `$GOPATH/bin` which is by default `$HOME/go/bin`.

## Build docker image

The [Makefile](Makefile) provides a target to build and push image to a Docker registry.  With a given release version `{VERSION}` and given Docker registry URL `{DOCKER_REGISTRY}`, one does the following command:

```bash
$ VERSION={VERSION} DOCKER_REGISTRY={DOCKER_REGISTRY} make docker-release
```

You will be prompted with login/password the docker registry authentication.  The resulting image will be tagged as `{DOCKER_REGISTRY}/dfe_runnerd:{VERSION}`.
