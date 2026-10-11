package config

import "testing"

func TestFactoryJudgmentConfigurationAdmission(t *testing.T) {
	base := Config{OperatorID: 1, DatabaseDSNFile: "/home/tester/secrets/database", HostSocket: "/home/tester/runtime/host"}
	for _, change := range []func(*Config){
		func(c *Config) { c.ForgejoGraphActorID = 2 },
		func(c *Config) { c.ForgejoGraphCredentialFile = "/home/tester/secrets/graph" },
		func(c *Config) { c.ForgejoGraphCredentialFile = "relative"; c.ForgejoGraphActorID = 2 },
		func(c *Config) {
			c.ForgejoGraphCredentialFile = "/home/tester/secrets/graph"
			c.ForgejoGraphActorID = 2
		},
		func(c *Config) { c.FactoryJudgmentCallBudget = -1 },
		func(c *Config) { c.FactoryJudgmentCallBudget = 100001 },
		func(c *Config) { c.TypeSafeAPIKeyFile = "relative" },
	} {
		c := base
		change(&c)
		if validateConfigPaths(c) == nil {
			t.Fatal("accepted incomplete graph authority or unbounded consultation configuration")
		}
	}
	uid := uint32(2000)
	c := base
	c.ForgejoGraphActorID = 2
	c.ForgejoGraphCredentialFile = "/home/tester/secrets/graph"
	c.ForgejoBackgroundSocket = "/home/tester/runtime/native"
	c.ForgejoBackgroundHostUID = &uid
	c.ForgejoBackgroundCredentialFile = "/home/tester/secrets/background"
	c.TypeSafeAPIKeyFile = "/home/tester/secrets/judgment"
	c.FactoryJudgmentCallBudget = 8
	if err := validateConfigPaths(c); err != nil {
		t.Fatal(err)
	}
}
