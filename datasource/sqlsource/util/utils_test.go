package util

import "testing"

func TestDescriptorToIdentifier(t *testing.T) {
	tables := []struct {
		x string
		y string
	}{
		{"person_personKey", "person_person_key"},
		{"person_organizationalUnit_personKey", "person_organizational_unit_person_key"},
	}

	for _, table := range tables {
		total := DescriptorToIdentifier(table.x)
		if total != table.y {
			t.Errorf("DescriptorToIdentifier(%s) was incorrect, got: %s, want: %s.", table.x, total, table.y)
		}
	}
}

func TestIdentifierToDescriptor(t *testing.T) {
	tables := []struct {
		x string
		y string
	}{
		{"person_key", "personKey"},
	}

	for _, table := range tables {
		total := IdentifierToDescriptor(table.x)
		if total != table.y {
			t.Errorf("IdentifierToDescriptor(%s) was incorrect, got: %s, want: %s.", table.x, total, table.y)
		}
	}
}
