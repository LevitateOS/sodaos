package factory

import (
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

// Sponsorship authorizes factory use of one provider connection for one
// repository and its assigned roles within an explicit allowance. It
// references the canonical broker grant, connection and credential
// generation; it never copies credential custody.
type Sponsorship struct {
	Roles            []string `json:"roles"`
	Repository       int64    `json:"repository,string"`
	Revision         int64    `json:"revision"`
	GrantedBy        int64    `json:"granted_by,string"`
	Generation       int64    `json:"generation"`
	Connection       string   `json:"connection"`
	GrantID          string   `json:"grant_id"`
	AllowanceMinutes int      `json:"allowance_minutes"`
	MaxConcurrent    int      `json:"max_concurrent"`
	Active           bool     `json:"active"`
}

func (s Sponsorship) Validate() error {
	if s.Repository <= 0 || s.Revision < 0 || s.GrantedBy <= 0 || s.Generation <= 0 {
		return errors.New("invalid sponsorship identity")
	}
	if s.Connection == "" || len(s.Connection) > 128 || s.GrantID == "" || len(s.GrantID) > 128 {
		return errors.New("invalid sponsorship connection reference")
	}
	if len(s.Roles) == 0 || len(s.Roles) > 2 {
		return errors.New("sponsorship must permit one or both factory roles")
	}
	seen := make(map[string]bool, len(s.Roles))
	for _, role := range s.Roles {
		if !project.ValidFactoryRole(role) || seen[role] {
			return errors.New("invalid sponsorship role")
		}
		seen[role] = true
	}
	if s.AllowanceMinutes < 1 || s.AllowanceMinutes > 10080 || s.MaxConcurrent < 1 || s.MaxConcurrent > 8 {
		return errors.New("invalid sponsorship allowance")
	}
	return nil
}
