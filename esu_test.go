package esu

import "testing"

func TestParseARN(t *testing.T) {
	cases := map[string]ARN{
		"family":     {"", "family", ""},
		"family:123": {"", "family", "123"},
		"family:456": {"", "family", "456"},
		"arn:aws:ecs:us-east-1:12345678:task-definition/family:456":     {"arn:aws:ecs:us-east-1:12345678:task-definition", "family", "456"},
		"arn:aws:ecs:us-east-1:12345678:container-instance/containerid": {"arn:aws:ecs:us-east-1:12345678:container-instance", "containerid", ""},
	}

	for s, e := range cases {
		if e.String() != s {
			t.Errorf("String() error, wanted %s, was %s", e, s)
		}
		a := ParseARN(s)
		if a != e {
			t.Errorf("Parse error, wanted %s was %s", e, a)
		}
	}
}
