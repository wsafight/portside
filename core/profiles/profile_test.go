package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"portside/core/porthome"
)

func TestCreateAndListPrefixes(t *testing.T) {
	home := setupHome(t)

	prefix, err := CreatePrefix("steam-main")
	if err != nil {
		t.Fatal(err)
	}
	if prefix.ID != "steam-main" {
		t.Fatalf("prefix id = %s", prefix.ID)
	}
	if prefix.Path != filepath.Join(home, "prefixes", "steam-main") {
		t.Fatalf("prefix path = %s", prefix.Path)
	}

	list, err := ListPrefixes()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "steam-main" {
		t.Fatalf("prefix list = %#v", list)
	}
}

func TestCreatePrefixRejectsInvalidID(t *testing.T) {
	setupHome(t)

	if _, err := CreatePrefix("Steam Main"); err == nil {
		t.Fatal("expected invalid id error")
	}
}

func TestAddReadAndListProfiles(t *testing.T) {
	setupHome(t)
	if _, err := CreatePrefix("steam-main"); err != nil {
		t.Fatal(err)
	}

	profile, err := AddProfile(AddOptions{
		ID:     "elden-ring",
		Name:   "Elden Ring",
		AppID:  1245620,
		Prefix: "steam-main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != "Elden Ring" {
		t.Fatalf("name = %s", profile.Name)
	}
	if profile.Run.EXE != "C:/Program Files (x86)/Steam/steam.exe" {
		t.Fatalf("run exe = %s", profile.Run.EXE)
	}
	if len(profile.Run.Args) != 2 || profile.Run.Args[0] != "-applaunch" || profile.Run.Args[1] != "1245620" {
		t.Fatalf("run args = %#v", profile.Run.Args)
	}
	if profile.Steam.AppManifest != "steamapps/appmanifest_1245620.acf" {
		t.Fatalf("app manifest = %s", profile.Steam.AppManifest)
	}

	read, err := ReadProfile("elden-ring")
	if err != nil {
		t.Fatal(err)
	}
	if read.ID != profile.ID || read.AppID != profile.AppID {
		t.Fatalf("read profile = %#v, want %#v", read, profile)
	}

	list, err := ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "elden-ring" {
		t.Fatalf("profile list = %#v", list)
	}
}

func TestAddProfileDefaultsNameAndRejectsBadInput(t *testing.T) {
	setupHome(t)
	if _, err := CreatePrefix("steam-main"); err != nil {
		t.Fatal(err)
	}

	profile, err := AddProfile(AddOptions{
		ID:     "test-game",
		AppID:  1,
		Prefix: "steam-main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != "test-game" {
		t.Fatalf("default name = %s", profile.Name)
	}

	if _, err := AddProfile(AddOptions{ID: "bad-appid", Prefix: "steam-main"}); err == nil {
		t.Fatal("expected appid error")
	}
	if _, err := AddProfile(AddOptions{ID: "missing-prefix", AppID: 1, Prefix: "other"}); err == nil {
		t.Fatal("expected missing prefix error")
	}
	if _, err := AddProfile(AddOptions{ID: "Bad ID", AppID: 1, Prefix: "steam-main"}); err == nil {
		t.Fatal("expected invalid id error")
	}
}

func TestListProfilesIgnoresNonJSONFiles(t *testing.T) {
	home := setupHome(t)
	if _, err := CreatePrefix("steam-main"); err != nil {
		t.Fatal(err)
	}
	if _, err := AddProfile(AddOptions{ID: "a-game", AppID: 1, Prefix: "steam-main"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(porthome.ProfilesDir(home), "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "a-game" {
		t.Fatalf("profile list = %#v", list)
	}
}

func setupHome(t *testing.T) string {
	t.Helper()

	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("PORTSIDE_HOME", home)
	if _, err := porthome.Init(); err != nil {
		t.Fatal(err)
	}
	return home
}
