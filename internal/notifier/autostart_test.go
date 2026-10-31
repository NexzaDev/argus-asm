package notifier

import (
	"path/filepath"
	"strings"
	"testing"
)

var testArgs = []string{"notifier", "run", "--db", "/home/user/data/argus-asm.db", "--interval", "5m0s", "--detail", "minimal"}

func TestPlanInstallPerOSPathsAreUserScoped(t *testing.T) {
	home := filepath.Join("/", "home", "user")
	cases := []struct {
		goos, wantSuffix string
	}{
		{"linux", filepath.Join(".config", "autostart", LinuxFileName)},
		{"darwin", filepath.Join("Library", "LaunchAgents", DarwinLabel+".plist")},
		{"windows", filepath.Join("Microsoft", "Windows", "Start Menu", "Programs", "Startup", WindowsFileName)},
	}
	for _, tc := range cases {
		plan, err := PlanInstall(Layout{GOOS: tc.goos, Home: home, AppData: filepath.Join(home, "AppData", "Roaming"), Executable: "/opt/argus-asm", Args: testArgs})
		if err != nil {
			t.Fatalf("%s: %v", tc.goos, err)
		}
		if !strings.HasSuffix(plan.Path, tc.wantSuffix) {
			t.Fatalf("%s path = %s", tc.goos, plan.Path)
		}
		if strings.Contains(plan.Path, "/usr/") || strings.Contains(plan.Path, "ProgramData") {
			t.Fatalf("%s path is system-wide: %s", tc.goos, plan.Path)
		}
	}
}

func TestPlanInstallContentRunsNotifierRun(t *testing.T) {
	plan, err := PlanInstall(Layout{GOOS: "linux", Home: "/home/user", Executable: "/opt/argus asm", Args: testArgs})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.Content, `Exec="/opt/argus asm" "notifier" "run"`) || !strings.Contains(plan.Content, "X-GNOME-Autostart-enabled=true") {
		t.Fatalf("linux content = %s", plan.Content)
	}
	plan, err = PlanInstall(Layout{GOOS: "darwin", Home: "/Users/user", Executable: "/opt/a&b", Args: testArgs})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.Content, "<string>/opt/a&amp;b</string>") || !strings.Contains(plan.Content, "<key>RunAtLoad</key>") {
		t.Fatalf("darwin content = %s", plan.Content)
	}
	plan, err = PlanInstall(Layout{GOOS: "windows", AppData: `C:\Users\u\AppData\Roaming`, Executable: `C:\Program Files\argus-asm.exe`, Args: testArgs})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.Content, `start "" /min "C:\Program Files\argus-asm.exe" "notifier" "run"`) {
		t.Fatalf("windows content = %s", plan.Content)
	}
}

func TestPlanInstallRejectsUnsafeValues(t *testing.T) {
	if _, err := PlanInstall(Layout{GOOS: "linux", Home: "/h", Executable: "/x\nExec=evil", Args: nil}); err == nil {
		t.Fatal("line break accepted")
	}
	if _, err := PlanInstall(Layout{GOOS: "windows", AppData: `C:\a`, Executable: `C:\x.exe`, Args: []string{`--db=C:\a"b`}}); err == nil {
		t.Fatal("quote accepted for Windows cmd")
	}
	if _, err := PlanInstall(Layout{GOOS: "plan9", Home: "/h", Executable: "/x"}); err == nil {
		t.Fatal("unsupported OS accepted")
	}
}
