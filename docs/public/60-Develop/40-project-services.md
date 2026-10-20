# Run Project development services

Project services are development databases and application containers inside the shared Project. A Project administrator manages persistent workloads; members connect only to assigned endpoints and data.

## Inspect and start a service

Work in a Project terminal as an administrator authorized to manage its nested Podman engine or systemd services. Keep Compose files and unit definitions with the Project or repository, and coordinate port use with other users. The Project engine is separate from the appliance engine.

From the directory containing `compose.yml`, inspect services and recent logs before explicitly starting one. Replace `SERVICE` with a service name defined in that Compose file:

```sh
podman compose -f compose.yml ps
podman compose -f compose.yml logs --tail=100 SERVICE
podman compose -f compose.yml up -d SERVICE
```

For a Project systemd unit, inspect its status and logs before starting it. Replace `SERVICE` with the unit name defined for the Project:

```sh
systemctl status SERVICE.service
journalctl -u SERVICE.service --no-pager -n 100
systemctl start SERVICE.service
```

After a successful start, the Compose service appears in `ps` or the systemd unit reports active; inspect logs and verify the endpoint before relying on the service. These commands manage development workloads inside the Project and do not deploy production applications.

## Connect to assigned endpoints

Members use only the service endpoints and credentials the administrator has made available. Bind listeners to the intended Project interface and coordinate address and port selection. Use an SSH local forward when a service should be reached through SSH instead of exposed on a shared interface. The placeholders below are the local client port, Project-local listener port, and Project login/address shown in access details:

```sh
ssh -L LOCAL_PORT:127.0.0.1:PROJECT_PORT USER@PROJECT_ADDRESS
```

Factory coder and reviewer roles also need explicit endpoint access, narrowly scoped credentials, and disposable test data. A prompt or readiness claim does not grant access by itself. Test service health and required permissions from the assigned account. Never run destructive tests or migrations against shared human data; changes to shared data require the appropriate Project authority.

## Preserve data and recheck readiness

Keep durable data in the service's intended Project volume or persistent path. Terminal End does not stop independent services or delete their volumes. Project Stop interrupts users and stops live workloads while preserving Project data.

After a restart or service replacement, check status and logs, then verify reachability and required permissions from the actual assigned account. Reconnecting a terminal does not establish service readiness. Factory resource assignment belongs to [Project preparation](../50-Spaces-and-Projects/30-prepare-project.md).

For a Canlang application preview, use [Canlang development](50-canlang-development.md). An independently installed private application has its own lifecycle in [Private applications](../70-Operate/40-private-applications.md).
