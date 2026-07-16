package claudecode

import "testing"

func TestCanonicalSkillName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "validate-json", want: "validate-json"},
		{name: "e2e-test-creator-v4.2", want: "e2e-test-creator-v4-2"},
		{name: " review skill/v2 ", want: "review-skill-v2"},
		{name: "Review_Skill-V2", want: "Review_Skill-V2"},
	}

	for _, test := range tests {
		if got := CanonicalSkillName(test.name); got != test.want {
			t.Errorf("CanonicalSkillName(%q) = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestSkillRegistryScopedName(t *testing.T) {
	if got := SkillRegistryScopedName("", "e2e-test-creator-v4.2"); got != "sdk-skill-registry:e2e-test-creator-v4-2" {
		t.Fatalf("SkillRegistryScopedName() = %q", got)
	}
	if got := SkillRegistryScopedName("custom-registry", "validate-json"); got != "custom-registry:validate-json" {
		t.Fatalf("SkillRegistryScopedName() = %q", got)
	}
}
