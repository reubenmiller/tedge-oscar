package flows

type InstanceStep struct {
	Script string `toml:"script"`
}

type InstanceInputMQTT struct {
	Topics []string `toml:"topics"`
}

type InstanceInput struct {
	MQTT InstanceInputMQTT `toml:"mqtt"`
}

type InstanceFile struct {
	Input InstanceInput  `toml:"input"`
	Steps []InstanceStep `toml:"steps"`
}
