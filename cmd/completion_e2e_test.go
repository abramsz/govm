package cmd

import (
	"strings"
	"testing"
)

func TestCompleteViaCobra(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	// Create a fake install for completion tests.
	ct.createFakeInstall(t, "1.23.4")

	t.Run("default_completion", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"__complete", "default", ""})
		if err != nil {
			t.Fatal(err)
		}
		// cobra's __complete output format: completions then ":<directive>"
		if !strings.Contains(stdout, "1.23.4") {
			t.Errorf("expected '1.23.4' in __complete output:\n%s", stdout)
		}
	})

	t.Run("use_completion", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"__complete", "use", ""})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stdout, "1.23.4") {
			t.Errorf("expected '1.23.4' in __complete output:\n%s", stdout)
		}
	})

	t.Run("uninstall_completion", func(t *testing.T) {
		stdout, _, err := ct.runCmd(t, []string{"__complete", "uninstall", ""})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stdout, "1.23.4") {
			t.Errorf("expected '1.23.4' in __complete output:\n%s", stdout)
		}
	})
}
