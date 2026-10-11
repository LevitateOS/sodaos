package factory

import "testing"

func validAllowance() PlanningAllowance {
	return PlanningAllowance{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Amount: 5, Unit: PlanningAllowanceUnit, Reserved: 1, Consumed: 2, Ceiling: 10, CreatedUnix: 1}
}

func TestPlanningAllowanceBoundsAndAvailability(t *testing.T) {
	a := validAllowance()
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	if a.Available() != 2 {
		t.Fatal("available miscomputed")
	}
	a.Reserved = 4
	if a.Validate() == nil {
		t.Fatal("overcommitted allowance accepted")
	}
	a = validAllowance()
	a.Amount = 11
	if a.Validate() == nil {
		t.Fatal("allowance above ceiling accepted")
	}
	a = validAllowance()
	a.Unit = "credits"
	if a.Validate() == nil {
		t.Fatal("billing unit accepted")
	}
}
