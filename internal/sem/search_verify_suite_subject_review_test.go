package sem

import "testing"

func TestSuiteSubjectCannotBeCapturedByUnrelatedManifest(t *testing.T) {
	for _, kind := range []string{"Cargo", "Maven", "Gradle", "Node", "Composer", "Ruby", "CMake"} {
		if searchVerifySuiteApplies(searchVerifySubject{sourcePath: "service.py"}, kind) {
			t.Errorf("Python source selected %s", kind)
		}
	}
	if !searchVerifySuiteApplies(searchVerifySubject{sourcePath: "service.py"}, "Pytest") {
		t.Fatal("Python not recognized")
	}
	if !searchVerifySuiteApplies(searchVerifySubject{sourcePath: "native.c", testPath: "wrapper.test.ts"}, "Node") {
		t.Fatal("covering test not recognized")
	}
}

func TestCargoSuiteRecognizesNativeBridgeHeaders(t *testing.T) {
	for _, extension := range []string{".hh", ".hpp", ".hxx"} {
		t.Run(extension, func(t *testing.T) {
			evidence := searchVerifyTestEvidenceWithout(map[string]string{
				"Cargo.toml": "[package]\nname = \"native_bridge\"\nversion = \"0.1.0\"\n",
			})
			command := deriveSearchVerifySuiteCommand(searchVerifySubject{sourcePath: "include/bridge" + extension}, &evidence)
			if command == nil || command.Command != "cargo test" {
				t.Fatalf("Cargo-only native bridge header: command = %+v, want cargo test", command)
			}
		})
	}
}
