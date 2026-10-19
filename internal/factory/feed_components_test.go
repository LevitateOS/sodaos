package factory

import "testing"

func TestFeedComponentAuthority(t *testing.T) {
	for _, test := range []struct {
		name    string
		id      string
		profile MCPProfile
		tool    MCPTool
		allowed bool
	}{
		{"builder plan change", "BU09", MCPProfileBuilder, "factory.suggest", true},
		{"inspector finding", "IN03", MCPProfileInspector, "factory.flag", true},
		{"shared question", "S02", MCPProfileArchitect, "factory.ask", true},
		{"local decision", "S16", MCPProfileFixer, "factory.record_decision", true},
		{"cross role finding", "IN03", MCPProfileBuilder, "factory.flag", false},
		{"question through progress", "S02", MCPProfileBuilder, "factory.partial", false},
		{"forged merge", "F06", MCPProfileBuilder, "factory.partial", false},
		{"forged verification", "F10", MCPProfileTestPilot, MCPToolFinishAppVerify, false},
		{"confirmed library mutation", "LA05", MCPProfileLibraryAdmin, "factory.remove_library_item", true},
		{"library mutation through progress", "LA05", MCPProfileLibraryAdmin, "factory.partial", false},
		{"cross role library mutation", "LA05", MCPProfileBuilder, "factory.remove_library_item", false},
		{"unknown worker shared outlet", "S04", MCPProfile("unknown"), "factory.partial", false},
		{"unknown component", "BU99", MCPProfileBuilder, "factory.partial", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateFeedComponent(test.id, test.profile, test.tool)
			if (err == nil) != test.allowed {
				t.Fatalf("allowed=%v, got %v", test.allowed, err)
			}
		})
	}
}
