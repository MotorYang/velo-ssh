package cmd

import "testing"

func TestParseRemoteSpec(t *testing.T) {
	spec, err := parseRemoteSpec("prod:/var/app/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Server != "prod" || spec.Path != "/var/app/file.txt" {
		t.Fatalf("spec = %#v", spec)
	}
}

func TestParseRemoteSpecRejectsInvalid(t *testing.T) {
	for _, value := range []string{"prod", ":/tmp/a", "prod:"} {
		if _, err := parseRemoteSpec(value); err == nil {
			t.Fatalf("expected %q to fail", value)
		}
	}
}
