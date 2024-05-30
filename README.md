# smartctl_ssacli_exporter
Export S.M.A.R.T. metrics from HP Smart Array disks. &amp; disk smartctl with auto detect disk

| Flag name              | Default Value    | Desc                                     |
|------------------------|------------------|------------------------------------------|
| web.listen-address     |:9633             | Exporter listener port && address        |
| web.telemetry-path     |/metrics          | URL path for surfacing collected metrics |
| smartctl.path          |/usr/bin/smartctl | Path to the smartctl executable          |
| ssacli.path            |/usr/bin/ssacli   | Path to the ssacli executable            |
| lsscsi.path            |/usr/bin/lsscsi   | Path to the lsscsi executable            |
| sudo.path              |/usr/bin/sudo     | Path to the sudo executable              |
| log.level              |info              | Filter for logging                       |

## Usage

**Prerequisites**
The `smartctl`, `ssacli`, and `lsscsi` utilities must be installed and available on the system. The paths to these executables can be provided by the associated command line arguments.

The user running `smartctl_ssacli_exporter` must have passwordless sudo authority to execute the `smartctl` and `ssacli` commands, or be root themselves. This is because `ssacli` must always be run as root, and `smartctl` must be run as root when interacting with SCSI devices. 

``` bash
./smartctl_ssacli_exporter
```

## Install

### Build from source
``` Bash
git clone https://github.com/john-craig/smartctl_ssacli_exporter.git
go get
go build
```

## Dashboard
Grafana ID: TBD