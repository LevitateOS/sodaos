package host

import (
	"context"
	"errors"
	"fmt"
	"github.com/levitateos/sodaos/internal/project"
	"os"
	"path/filepath"
)

// SetMemberAccessFence prevents automatic and explicit Project start while
// native member effects are uncertain. It is published before the DB hold and
// remains after an interrupted request; retry reconciles the same member.
func SetMemberAccessFence(pid string, uid int64, held bool) error {
	if !project.ValidID(pid) || uid <= 0 {
		return errors.New("invalid member start fence")
	}
	root := "/var/lib/soda/dashboard/member-access"
	dir := filepath.Join(root, pid)
	if held {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		file, err := os.OpenFile(filepath.Join(dir, fmt.Sprintf("%d", uid)), os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		if err = file.Sync(); err != nil {
			file.Close()
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
	} else {
		if err := os.Remove(filepath.Join(dir, fmt.Sprintf("%d", uid))); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	parent, err := os.Open(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if parent != nil {
		err = parent.Sync()
		_ = parent.Close()
		if err != nil {
			return err
		}
	}
	if !held { // A concurrent member's fence keeps the directory and boot hold.
		_ = os.Remove(dir)
	}
	parent, err = os.Open(root)
	if err != nil {
		return err
	}
	defer parent.Close()
	return parent.Sync()
}

func (c *Client) MemberAccess(ctx context.Context, in project.MemberAccess) (project.MemberAccessState, error) {
	var out project.MemberAccessState
	if err := in.Validate(); err != nil {
		return out, err
	}
	err := c.callLimit(ctx, "/member-access", in, &out, 65536)
	if err == nil && (out.Project != in.Project || out.Login != in.Login || out.Identity != in.Identity || out.Keys == nil || out.Processes == nil || out.Revision < 0) {
		err = errors.New("native member access identity was not confirmed")
	}
	if err == nil && in.Action != "inspect" && (out.Revision != in.Revision || !out.LoginDisabled || !out.KeysRemoved || !out.ProcessesStopped || (in.Action == "revoke" && !out.Held) || (in.Action == "restore" && (out.Held || !out.RequiresJoin))) {
		err = errors.New("native member access effects were not confirmed")
	}
	return out, err
}
