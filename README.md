# Dynamore feature extraction runner

A small deamon that submits HPC job to run feature extraction upon receiving payload message from a Redis server.

## Installation

Download the RPM package from the release assets, and run

```bash
$ yum localinstall dynamore-feature-extraction-runner-0.1.0-1.el7.x86_64.rpm
```

A daemon called `dfe_runnerd` will be enabled and started via systemd.

## Configuration

Uncomment and change the variables in the file is located in `/etc/ysconfig/dfe_runnerd`.  Restart the daemon after changing the values:

```bash
$ systemctl restart dfe_runnerd
```
