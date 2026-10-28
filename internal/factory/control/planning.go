package control

import (
	"context"

	"github.com/levitateos/sodaos/internal/factory"
)

type ProductJudgment interface {
	Select(context.Context, any, string, map[string]any) (factory.PriorityAnswer, error)
}

// PlanningJudgments carries the System One chooser the JEV purposes share.
type PlanningJudgments struct {
	Chooser ProductJudgment
}
