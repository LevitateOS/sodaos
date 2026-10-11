package factory

import "testing"

func TestDiffIntentTasksSeparatesReusedChangedAddedRemoved(t *testing.T) {
	old := validIntent()
	current := validIntent()
	current.ID = "d666666666666666666666666"
	current.Predecessor = old.ID
	current.Tasks[0].Objective = "Create accessible request form"
	current.Tasks = append(current.Tasks[:1], IntentTask{Key: "d777777777777777777777777", Objective: "Notify managers", Outcomes: []string{"Managers are notified"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{"d444444444444444444444444"}})
	d, err := DiffIntentTasks(old, current)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Reused) != 0 || len(d.Changed) != 1 || d.Changed[0] != "d444444444444444444444444" {
		t.Fatalf("changed %+v", d)
	}
	if len(d.Added) != 1 || d.Added[0] != "d777777777777777777777777" {
		t.Fatalf("added %+v", d)
	}
	if len(d.Removed) != 1 || d.Removed[0] != "d555555555555555555555555" {
		t.Fatalf("removed %+v", d)
	}
	if !d.EdgesChanged {
		t.Fatal("edge change missed")
	}
	if _, err = DiffIntentTasks(old, IntentRevision{ID: "dx", Intent: "d999999999999999999999999"}); err == nil {
		t.Fatal("cross-intent delta accepted")
	}
}
