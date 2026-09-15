package hostproject

// Config holds the project-network image and bridge fields needed to create and
// inspect environments. Tailnet companion settings stay on the host Daemon.
type Config struct {
	Image   string
	Network string
	Subnet  string
	Bridge  string
}
