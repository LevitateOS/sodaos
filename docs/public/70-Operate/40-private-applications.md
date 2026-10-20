# Operate a private application

Private Canlang applications have their own storage, accounts and service lifetime. An operator registers and installs a compatible application image; an authorized app administrator or operator then inspects it, explicitly starts it, or confirms a stop.

Build and provision a compatible Canlang bundle and image in the Project using [Canlang development](../60-Develop/50-canlang-development.md). Prepare app-local owner/member accounts while the application is stopped. Forgejo users, Project members and AI provider owners do not automatically become application users. The operator registers the exact loaded image and its separate app data identity, then chooses Install. Installation validates the image and storage references, configures its private route and leaves the application stopped.

Configure a separate private application hostname, DNS and client TLS trust as described in [Private networking and TLS](../30-Set-up/20-private-networking-and-tls.md). App cookies and authorization remain on that hostname. Once the intended client can reach the origin, select Start and confirm readiness, then sign in using the app's own account. A ready backend alone does not prove client DNS, routing or TLS trust.

Replace an image only with a compatible release that uses the same data identity and passes storage compatibility checks. Replacement stops the old generation, installs the new one and leaves it stopped; explicitly Start it after inspection. Schema migration and using an older image to undo database changes are unsupported. If an action's outcome is uncertain, inspect the recorded generation and service state before repeating it. Factory merge does not install or deploy an application.

To stop the app, choose Stop and confirm the effect. Stop disables boot startup and retains application data. It is independent of Project Stop: a stopped development Project does not stop the application, and opening the Applications view or linking an app to a Project does not start it. Optional Project association changes navigation only; it does not move storage, change the app's lifetime or deploy Project code.

For a stopped app backup or restore, follow the [application snapshot procedure](60-backups-and-restoration.md#private-application-snapshots). An existing application record cannot be silently retargeted to restored data.
