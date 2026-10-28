package skill

import (
	"strings"
	"testing"
)

func validate(t *testing.T, folder, content string) ([]string, error) {
	t.Helper()
	s, err := Load(writeSkill(t, folder, content))
	if err != nil {
		t.Fatal(err)
	}
	return Validate(s)
}

func TestValidateOK(t *testing.T) {
	warnings, err := validate(t, "pdf", "---\nname: pdf\ndescription: Work with PDFs.\nextra: kept\n---\n")
	if err != nil || len(warnings) != 0 {
		t.Errorf("got %v, %v", warnings, err)
	}
}

func TestValidateErrors(t *testing.T) {
	cases := map[string]struct{ content, want string }{
		"no name":         {"---\ndescription: d\n---\n", "name is missing"},
		"empty name":      {"---\nname: ''\ndescription: d\n---\n", "name must be a non-empty string"},
		"name not string": {"---\nname: [a]\ndescription: d\n---\n", "name must be a non-empty string"},
		"uppercase name":  {"---\nname: My-Skill\ndescription: d\n---\n", "invalid name"},
		"name with space": {"---\nname: my skill\ndescription: d\n---\n", "invalid name"},
		"leading dash":    {"---\nname: -x\ndescription: d\n---\n", "invalid name"},
		"no description":  {"---\nname: x\n---\n", "description is missing"},
		"blank desc":      {"---\nname: x\ndescription: '  '\n---\n", "description must be a non-empty string"},
		"empty file":      {"---\n---\n", "name is missing"},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			_, err := validate(t, "x", c.content)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), FileName) {
				t.Errorf("got %v, want %q with path", err, c.want)
			}
		})
	}
}

func TestValidateReportsAllErrors(t *testing.T) {
	_, err := validate(t, "x", "---\nlicense: MIT\n---\n")
	if err == nil || !strings.Contains(err.Error(), "name is missing") || !strings.Contains(err.Error(), "description is missing") {
		t.Errorf("got %v", err)
	}
}

func TestValidateWarnings(t *testing.T) {
	warnings, err := validate(t, "other-folder", "---\nname: pdf\ndescription: "+strings.Repeat("a", 1025)+"\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(warnings, "\n")
	if len(warnings) != 2 || !strings.Contains(all, `folder name "other-folder" differs`) || !strings.Contains(all, "longer than 1024") {
		t.Errorf("got %v", warnings)
	}
}

func TestValidateName(t *testing.T) {
	for _, ok := range []string{"pdf", "db-migration", "3d"} {
		if err := ValidateName(ok); err != nil {
			t.Errorf("ValidateName(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", "PDF", "-pdf", "a/b", "..", "my skill", strings.Repeat("a", maxNameLen+1)} {
		if err := ValidateName(bad); err == nil {
			t.Errorf("ValidateName(%q) passed", bad)
		}
	}
}
