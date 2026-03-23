//go:build integration

package plugin_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"gopkg.in/yaml.v3"
)

// refRe matches {{ namespace.key }} template expressions.
var refRe = regexp.MustCompile(`\{\{\s*(config|outputs|pushed|instance)\.([\w]+)\s*\}\}`)

// TestPluginSpecsIntegration validates that every template reference used in
// step commands and values is actually declared (config key, capture key, or
// push key) within the same spec.  instance.* references are always valid.
func TestPluginSpecsIntegration(t *testing.T) {
	var paths []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "plugin.yaml" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no plugin.yaml files found")
	}
	for _, p := range paths {
		p := p
		t.Run(p, func(t *testing.T) {
			validateTemplateRefs(t, p)
		})
	}
}

func validateTemplateRefs(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var spec pluginSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Build lookup sets from declarations.
	configKeys := make(map[string]bool)
	for k := range spec.Config {
		configKeys[k] = true
	}

	captureKeys := make(map[string]bool)
	pushKeys := make(map[string]bool)
	for _, s := range spec.Local.Provision {
		for k := range s.Capture {
			captureKeys[k] = true
		}
		if s.Type == "push" && s.Key != "" {
			pushKeys[s.Key] = true
		}
	}

	checkRefs := func(loc, text string) {
		for _, m := range refRe.FindAllStringSubmatch(text, -1) {
			ns, key := m[1], m[2]
			switch ns {
			case "config":
				if !configKeys[key] {
					t.Errorf("%s: {{ config.%s }} not declared in config", loc, key)
				}
			case "outputs":
				if !captureKeys[key] {
					t.Errorf("%s: {{ outputs.%s }} not captured in local.provision", loc, key)
				}
			case "pushed":
				if !pushKeys[key] {
					t.Errorf("%s: {{ pushed.%s }} not pushed in local.provision", loc, key)
				}
				// instance.* is populated by the runtime; no declaration required.
			}
		}
	}

	walkSteps := func(phase string, steps []step) {
		for i, s := range steps {
			loc := fmt.Sprintf("%s[%d]", phase, i)
			checkRefs(loc+".run", s.Run)
			checkRefs(loc+".value", s.Value)
			checkRefs(loc+".url", s.URL)
			for k, v := range s.Env {
				checkRefs(fmt.Sprintf("%s.env.%s", loc, k), v)
			}
		}
	}

	walkSteps("remote.install", spec.Remote.Install)
	walkSteps("remote.configure", spec.Remote.Configure)
	walkSteps("remote.start", spec.Remote.Start)
	walkSteps("remote.stop", spec.Remote.Stop)
	walkSteps("remote.health.steps", spec.Remote.Health.Steps)
	walkSteps("local.provision", spec.Local.Provision)
	walkSteps("local.deprovision", spec.Local.Deprovision)
}
