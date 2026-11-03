# Forge and Factory process lifecycle

`Run(Forge)` serves `cmd/soda-dashboard`; `Run(Factory)` serves
`cmd/soda-factory-server`. The shared lifecycle owns configuration, the store,
signals, Unix listeners, HTTP admission and shutdown. Each command runs one
product. Only Factory binds worker MCP, starts the coordinator and acquires its
ledger lock. Each product has its own operator endpoint.

The Factory Project control listener admits only the service's native OS UID.
The Factory operator listener admits the separately configured operator UID.
Extension requests retain native Fountain admission; worker requests retain
their run capabilities. These listeners cannot substitute for one another.

The integration tests include actual child processes against disposable
PostgreSQL, independent shutdown/restart, operator separation and reconnects.
Supply the fixture environment described in
[testing](../../../docs/development/testing.md).
