package identity

const MuseLaunchSocket = "/run/soda-muse-interface/launch.sock"

type (
	NestedRegistration struct {
		ChildID        string `json:"child_id"`
		ActorID        int64  `json:"actor_id,string"`
		RegistrationID string `json:"registration_id"`
		Muse           bool   `json:"muse"`
	}
	// LaunchRequest carries invocation preferences, never caller authority or credentials.
	LaunchRequest struct {
		Home         string              `json:"home,omitempty"`
		Register     *NestedRegistration `json:"register,omitempty"`
		ConfigHome   string              `json:"config_home,omitempty"`
		Term         string              `json:"term,omitempty"`
		ConnectionID string              `json:"connection_id"`
		CWD          string              `json:"cwd"`
		Args         []string            `json:"args"`
		TTY          bool                `json:"tty"`
		Cols         uint16              `json:"cols"`
		Rows         uint16              `json:"rows"`
	}
)

// LaunchControl is the bounded live shell control protocol.
type (
	LaunchControl struct {
		Signal int    `json:"signal,omitempty"`
		Cols   uint16 `json:"cols,omitempty"`
		Rows   uint16 `json:"rows,omitempty"`
	}
	LaunchExit struct {
		Code  int    `json:"code"`
		Error string `json:"error,omitempty"`
	}
)
