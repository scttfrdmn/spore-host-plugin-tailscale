package plugin_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// pluginSpec mirrors the subset of plugin.yaml fields we validate.
type pluginSpec struct {
	Name        string                 `yaml:"name"`
	Version     string                 `yaml:"version"`
	Description string                 `yaml:"description"`
	Config      map[string]configParam `yaml:"config"`
	Conditions  struct {
		Local  []condition `yaml:"local"`
		Remote []condition `yaml:"remote"`
	} `yaml:"conditions"`
	Local struct {
		Provision   []step `yaml:"provision"`
		Deprovision []step `yaml:"deprovision"`
	} `yaml:"local"`
	Remote struct {
		Install   []step  `yaml:"install"`
		Configure []step  `yaml:"configure"`
		Start     []step  `yaml:"start"`
		Stop      []step  `yaml:"stop"`
		Health    health  `yaml:"health"`
	} `yaml:"remote"`
}

type configParam struct {
	Required bool        `yaml:"required"`
	Default  interface{} `yaml:"default"`
	Type     string      `yaml:"type"`
}

type condition struct {
	Type    string `yaml:"type"`
	Run     string `yaml:"run"`
	OS      string `yaml:"os"`
	Message string `yaml:"message"`
}

type step struct {
	Type       string            `yaml:"type"`
	Run        string            `yaml:"run"`
	URL        string            `yaml:"url"`
	Dest       string            `yaml:"dest"`
	Src        string            `yaml:"src"`
	Key        string            `yaml:"key"`
	Value      string            `yaml:"value"`
	Background bool              `yaml:"background"`
	Capture    map[string]string `yaml:"capture"`
	Env        map[string]string `yaml:"env"`
}

type health struct {
	Interval string `yaml:"interval"`
	Steps    []step `yaml:"steps"`
}

func TestPluginSpecs(t *testing.T) {
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
			validateSpec(t, p)
		})
	}
}

func validateSpec(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var spec pluginSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	if spec.Name == "" {
		t.Error("name is empty")
	}
	if spec.Version == "" {
		t.Error("version is empty")
	} else if !isSemVer(spec.Version) {
		t.Errorf("version %q is not semver (expected X.Y.Z)", spec.Version)
	}
	if spec.Description == "" {
		t.Error("description is empty")
	}

	// Optional config params must have a default.
	for k, p := range spec.Config {
		if !p.Required && p.Default == nil {
			t.Errorf("config.%s: optional param has no default", k)
		}
	}

	// Conditions must have type and message.
	for i, c := range spec.Conditions.Local {
		if c.Type == "" {
			t.Errorf("conditions.local[%d]: missing type", i)
		}
		if c.Message == "" {
			t.Errorf("conditions.local[%d]: missing message", i)
		}
	}
	for i, c := range spec.Conditions.Remote {
		if c.Type == "" {
			t.Errorf("conditions.remote[%d]: missing type", i)
		}
		if c.Message == "" {
			t.Errorf("conditions.remote[%d]: missing message", i)
		}
	}

	// Remote install and start must be present for a remote plugin.
	if len(spec.Remote.Install) == 0 {
		t.Error("remote.install: no steps defined")
	}
	if len(spec.Remote.Start) == 0 {
		t.Error("remote.start: no steps defined")
	}

	// All steps must be well-formed.
	checkSteps(t, "remote.install", spec.Remote.Install)
	checkSteps(t, "remote.configure", spec.Remote.Configure)
	checkSteps(t, "remote.start", spec.Remote.Start)
	checkSteps(t, "remote.stop", spec.Remote.Stop)
	checkSteps(t, "remote.health.steps", spec.Remote.Health.Steps)
	checkSteps(t, "local.provision", spec.Local.Provision)
	checkSteps(t, "local.deprovision", spec.Local.Deprovision)
}

func checkSteps(t *testing.T, phase string, steps []step) {
	t.Helper()
	for i, s := range steps {
		if s.Type == "" {
			t.Errorf("%s[%d]: missing type", phase, i)
			continue
		}
		switch s.Type {
		case "run":
			if s.Run == "" {
				t.Errorf("%s[%d]: run step has no run command", phase, i)
			}
		case "fetch":
			if s.URL == "" {
				t.Errorf("%s[%d]: fetch step missing url", phase, i)
			}
			if s.Dest == "" {
				t.Errorf("%s[%d]: fetch step missing dest", phase, i)
			}
		case "extract":
			if s.Src == "" {
				t.Errorf("%s[%d]: extract step missing src", phase, i)
			}
			if s.Dest == "" {
				t.Errorf("%s[%d]: extract step missing dest", phase, i)
			}
		case "push":
			if s.Key == "" {
				t.Errorf("%s[%d]: push step missing key", phase, i)
			}
			if s.Value == "" {
				t.Errorf("%s[%d]: push step missing value", phase, i)
			}
		default:
			t.Errorf("%s[%d]: unknown step type %q", phase, i, s.Type)
		}
	}
}

// isSemVer returns true if v is a bare X.Y.Z semver string (no leading "v").
func isSemVer(v string) bool {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		// Allow pre-release suffix on the patch segment (e.g. "0-alpha").
		core := strings.SplitN(p, "-", 2)[0]
		if core == "" {
			return false
		}
		for _, c := range core {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
