package docker

import (
	"strings"
	"testing"
)

func TestGenerateDockerfileEdgeCasesNew(t *testing.T) {
	baseArgs := []interface{}{
		"latest",                     // imageVersion
		"1.0",                        // schemaVersion
		"127.0.0.1",                  // hostIP
		"11435",                      // ollamaPortNum
		"127.0.0.1:9090",             // kdepsHost
		"ARG FOO=bar",                // argsSection
		"ENV BAR=baz",                // envsSection
		"RUN apt-get install -y gcc", // pkgSection
		"",                           // pythonPkgSection
		"",                           // condaPkgSection
		"2024.10-1",                  // anacondaVersion
		"0.28.1",                     // pklVersion
		"UTC",                        // timezone
		"8080",                       // exposedPort
	}

	t.Run("devBuildMode", func(t *testing.T) {
		params := append(baseArgs, true /* installAnaconda */, true /* devBuildMode */, true /* apiServerMode */, false /* useLatest */)
		dockerfile := generateDockerfile(params[0].(string), params[1].(string), params[2].(string), params[3].(string), params[4].(string), params[5].(string), params[6].(string), params[7].(string), params[8].(string), params[9].(string), params[10].(string), params[11].(string), params[12].(string), params[13].(string), params[14].(bool), params[15].(bool), params[16].(bool), params[17].(bool))

		// Expect copy of kdeps binary due to devBuildMode true
		if !strings.Contains(dockerfile, "cp /cache/kdeps /bin/kdeps") {
			t.Fatalf("expected dev build copy step, got:\n%s", dockerfile)
		}
		// Anaconda installer should be present because installAnaconda true
		if !strings.Contains(dockerfile, "anaconda-linux-") {
			t.Fatalf("expected anaconda install snippet")
		}
		// Should expose port 8080 because apiServerMode true
		if !strings.Contains(dockerfile, "EXPOSE 8080") {
			t.Fatalf("expected EXPOSE directive")
		}
	})

	t.Run("prodBuildMode", func(t *testing.T) {
		params := append(baseArgs, false /* installAnaconda */, false /* devBuildMode */, false /* apiServerMode */, false /* useLatest */)
		dockerfile := generateDockerfile(params[0].(string), params[1].(string), params[2].(string), params[3].(string), params[4].(string), params[5].(string), params[6].(string), params[7].(string), params[8].(string), params[9].(string), params[10].(string), params[11].(string), params[12].(string), "", params[14].(bool), params[15].(bool), params[16].(bool), params[17].(bool))

		// Should pull kdeps via curl (not copy) because devBuildMode false
		if !strings.Contains(dockerfile, "raw.githubusercontent.com") {
			t.Fatalf("expected install kdeps via curl in prod build")
		}
		// Should not contain EXPOSE when apiServerMode false
		if strings.Contains(dockerfile, "EXPOSE") {
			t.Fatalf("did not expect EXPOSE directive when apiServerMode false")
		}
	})
}
