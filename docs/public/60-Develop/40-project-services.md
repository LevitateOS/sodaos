# Project services

Project services are development databases and application containers that run inside the shared Project. A Project administrator manages their persistent workloads; members connect only to endpoints and data assigned to them.

Work from a Project terminal as an administrator authorized to manage its nested Podman engine or systemd services. Keep Compose files and unit definitions with the Project or repository, and coordinate ports with other users. The Project engine is separate from the appliance engine. For example, from the directory containing `compose.yml`, inspect its services and recent logs, then explicitly start a service:

```sh
podman compose -f compose.yml ps
podman compose -f compose.yml logs --tail=100 SERVICE
podman compose -f compose.yml up -d SERVICE
```

For a Project systemd unit, inspect its state and logs before starting it:

```sh
systemctl status SERVICE.service
journalctl -u SERVICE.service --no-pager -n 100
systemctl start SERVICE.service
```

Replace `SERVICE` with the service or unit name defined for that Project. These commands inspect or start development workloads; they do not deploy production applications.

Members connect to service endpoints that the administrator has made available and use only the credentials and data assigned to them. Bind listeners to the intended Project interface and coordinate address and port selection; use an SSH local forward when a service should be reached through SSH instead of exposed on a shared interface. For example, forward a Project-local listener to a local client port:

```sh
ssh -L LOCAL_PORT:127.0.0.1:PROJECT_PORT USER@PROJECT_ADDRESS
```

Use the Project login and address shown in its access details. Factory coder and reviewer roles also need explicit endpoint access, narrowly scoped credentials and disposable test data. A prompt or readiness claim does not grant access by itself. Test service health and the required permissions from the assigned account. Never run destructive tests or migrations against shared human data; service administration and changes to shared data require the appropriate Project authority.

Keep durable data in the service's intended Project volume or persistent path. Terminal End does not stop independent services or delete their volumes. Project Stop interrupts users and stops live workloads while preserving Project data. After a restart or service replacement, check service status and logs, then verify reachability and required permissions from the actual assigned account; reconnecting a terminal does not establish readiness.

Factory resource assignment belongs to [Project preparation](../50-Spaces-and-Projects/30-prepare-project.md). For a Canlang application preview, use [Canlang development](50-canlang-development.md); an independently installed private application has its own lifecycle in [Private applications](../70-Operate/40-private-applications.md).
