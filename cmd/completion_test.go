package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCompleteInstalledVersions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOVM_HOME", dir)
	ct := &cmdTest{govmHome: dir}

	// With no installs, it should return empty.
	versions, directive := completeInstalledVersions(&cobra.Command{}, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp, got %d", directive)
	}
	if len(versions) != 0 {
		t.Errorf("expected empty, got %v", versions)
	}

	// Create fake installs.
	ct.createFakeInstall(t, "1.23.4")
	ct.createFakeInstall(t, "1.25.11")

	// Should return installed versions, newest first.
	versions, directive = completeInstalledVersions(&cobra.Command{}, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp, got %d", directive)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %v", versions)
	}
	if versions[0] != "1.25.11" {
		t.Errorf("expected newest first, got %v", versions)
	}
}
