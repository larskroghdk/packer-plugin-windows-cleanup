package main

import (
	"encoding/json"
	"os/exec"
	"testing"
)

func TestPluginRegistration(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "describe").Output()
	if err != nil {
		t.Fatal(err)
	}
	var description struct {
		Provisioners []string `json:"provisioners"`
		APIVersion   string   `json:"api_version"`
	}
	if err := json.Unmarshal(out, &description); err != nil {
		t.Fatal(err)
	}
	if len(description.Provisioners) != 1 || description.Provisioners[0] != "-packer-default-plugin-name-" || description.APIVersion == "" {
		t.Fatalf("%s", out)
	}
}
