package tests

import (
	"testing"

	"../src/gamelist"
)

func TestGetProfileName(t *testing.T) {
	tables := []struct {
		s           string
		profileName string
		isVanity    bool
	}{
		{"https://steamcommunity.com/id/murgonen/videos/", "murgonen", true},
		{"https://steamcommunity.com/id/murgonen", "murgonen", true},
		{"https://steamcommunity.com/id/murgonen?xml=1", "murgonen", true},
		{"https://steamcommunity.com/profiles/76561198018467980?xml=1", "76561198018467980", false},
		{"https://steamcommunity.com/profiles/76561198018467980", "76561198018467980", false},
		{"https://steamcommunity.com/profiles/76561198018467980/games/?tab=all", "76561198018467980", false},
		{"murgonen", "murgonen", true},
		{"12341234", "12341234", true},
		{"12341234123412341", "12341234123412341", false},
		{"https://xsteamcommunity.com/id/murgonen/videos/", "https://xsteamcommunity.com/id/murgonen/videos/", true},
	}

	for _, table := range tables {
		profileName, isVanity := gamelist.GetProfileName(table.s)
		if profileName != table.profileName {
			t.Errorf("Invalid profile name for %s, got %s, want %s.", table.s, profileName, table.profileName)
		}

		if isVanity != table.isVanity {
			t.Errorf("Invalid vanity value for %v, got %v, want %v.", table.s, isVanity, table.isVanity)
		}
	}
}
