package factory

import (
	"errors"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

// PlanningAllowanceUnit is the explicit real unit for design consultations.
// No billing-unit conversion; execution funding stays separate.
const PlanningAllowanceUnit = "planning_calls"

// PlanningAllowance is an independently revisioned finite consultation budget
// for one intent. It carries its own predecessor CAS chain, separate from
// intent revisions: raising allowance alone preserves effective scope, native
// issue mappings and prior consumption. Reserved holds in-flight calls;
// Consumed is cumulative spent including failures; Available is Amount minus
// both. An interrupted reservation without a durable answer stays charged.
type PlanningAllowance struct {
	ID          string `json:"id"`
	Intent      string `json:"intent"`
	Predecessor string `json:"predecessor,omitempty"`
	Owner       int64  `json:"owner,string"`
	Amount      int    `json:"amount"`
	Unit        string `json:"unit"`
	Reserved    int    `json:"reserved"`
	Consumed    int    `json:"consumed"`
	Ceiling     int    `json:"ceiling"`
	Effect      string `json:"effect,omitempty"`
	CreatedUnix int64  `json:"created_unix"`
}

// Available returns remaining calls that can still be reserved.
func (a PlanningAllowance) Available() int {
	available := a.Amount - a.Reserved - a.Consumed
	if available < 0 {
		return 0
	}
	return available
}

func (a PlanningAllowance) Validate() error {
	if !project.ValidDecisionID(a.ID) || !project.ValidDecisionID(a.Intent) || a.Owner <= 0 || a.CreatedUnix <= 0 {
		return errors.New("invalid allowance identity")
	}
	if a.Predecessor != "" && !project.ValidDecisionID(a.Predecessor) {
		return errors.New("invalid allowance predecessor")
	}
	if a.Unit != PlanningAllowanceUnit {
		return errors.New("invalid allowance unit")
	}
	if a.Amount < 0 || a.Amount > 100 || a.Ceiling < 0 || a.Ceiling > 100 || a.Amount > a.Ceiling {
		return errors.New("invalid allowance bounds")
	}
	if a.Reserved < 0 || a.Consumed < 0 || a.Reserved+a.Consumed > a.Amount {
		return errors.New("allowance overcommitted")
	}
	if a.Effect != "" && (strings.TrimSpace(a.Effect) == "" || len(a.Effect) > 4000) {
		return errors.New("invalid allowance effect")
	}
	return nil
}
