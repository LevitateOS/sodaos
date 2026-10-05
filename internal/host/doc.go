// Package host is the privileged Unix client surface for the Rust
// `soda-host` daemon. It keeps the Go client (RPC, terminal streams,
// preparation/factory/tailnet calls) and the terminal wire types the web
// layer needs. The daemon server (mux, admission, dispatch, config load)
// and all executors (project, terminal, tailnet) are Rust; the Go
// predecessors were removed at the executor cutover. It is not the
// SQLite owner, browser OAuth surface or build/release controller.
package host
