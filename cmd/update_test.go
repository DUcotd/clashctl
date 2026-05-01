package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateCommandProvidesSelfAlias(t *testing.T) {
	if !hasAlias(updateCmd, "self") {
		t.Fatal("update command should provide the self alias")
	}
}

func TestFinishUpdateReportStoresError(t *testing.T) {
	prev := updateJSON
	t.Cleanup(func() { updateJSON = prev })
	updateJSON = false
	report := &updateRunReport{CurrentVersion: "v1.0.0", Action: "check"}
	err := finishReport(report, errors.New("boom"), false)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("finishReport() error = %v", err)
	}
	if report.Error != "boom" {
		t.Fatalf("report.Error = %q, want boom", report.Error)
	}
}

func TestValidateDownloadedClashctlBinary(t *testing.T) {
	tmp := t.TempDir()
	good := filepath.Join(tmp, "clashctl-good")
	silent := filepath.Join(tmp, "clashctl-silent")
	broken := filepath.Join(tmp, "clashctl-broken")

	writeExecutableFile(t, good, "#!/bin/sh\necho 'clashctl v9.9.9'\n")
	writeExecutableFile(t, silent, "#!/bin/sh\nexit 0\n")
	writeExecutableFile(t, broken, "#!/bin/sh\necho 'boom' >&2\nexit 1\n")

	if err := validateDownloadedClashctlBinary(good); err != nil {
		t.Fatalf("validateDownloadedClashctlBinary(good) error = %v", err)
	}
	if err := validateDownloadedClashctlBinary(silent); err == nil {
		t.Fatal("validateDownloadedClashctlBinary(silent) should fail")
	}
	if err := validateDownloadedClashctlBinary(broken); err == nil {
		t.Fatal("validateDownloadedClashctlBinary(broken) should fail")
	}
}

func TestReplaceCurrentExecutableDoesNotTouchLegacyBakPath(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "clashctl")
	src := filepath.Join(dir, "clashctl-new")
	legacyBackup := dest + ".bak"

	writeExecutableFile(t, dest, "#!/bin/sh\nif [ \"$1\" = \"version\" ]; then\n  echo 'clashctl v1.0.0'\n  exit 0\nfi\necho old\n")
	writeExecutableFile(t, src, "#!/bin/sh\nif [ \"$1\" = \"version\" ]; then\n  echo 'clashctl v2.0.0'\n  exit 0\nfi\necho new\n")
	if err := os.WriteFile(legacyBackup, []byte("keep backup"), 0644); err != nil {
		t.Fatalf("WriteFile(legacy backup) error = %v", err)
	}

	if err := replaceCurrentExecutable(src, dest); err != nil {
		t.Fatalf("replaceCurrentExecutable() error = %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile(dest) error = %v", err)
	}
	if string(data) != "#!/bin/sh\nif [ \"$1\" = \"version\" ]; then\n  echo 'clashctl v2.0.0'\n  exit 0\nfi\necho new\n" {
		t.Fatalf("dest content = %q", string(data))
	}

	backupData, err := os.ReadFile(legacyBackup)
	if err != nil {
		t.Fatalf("ReadFile(legacy backup) error = %v", err)
	}
	if string(backupData) != "keep backup" {
		t.Fatalf("legacy backup content = %q", string(backupData))
	}

	matches, err := filepath.Glob(filepath.Join(dir, "clashctl.bak-*"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary backup files should be removed, got %v", matches)
	}
}

func writeExecutableFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
