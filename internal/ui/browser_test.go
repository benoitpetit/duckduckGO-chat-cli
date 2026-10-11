package ui

import "testing"

func TestBrowserOptionDescriptions(t *testing.T) {
	tests := []struct {
		value string
		mode  browseMode
		want  string
	}{
		{value: "[..]", mode: BrowseFiles, want: "parent folder"},
		{value: "[DIR] Pictures", mode: BrowseFiles, want: "open folder"},
		{value: "[FILE] photo.jpg", mode: BrowseFiles, want: "select file"},
		{value: "[Select current directory]", mode: BrowseDirectories, want: "use this folder"},
		{value: "photo.jpg", mode: BrowseFiles, want: ""},
	}
	for _, test := range tests {
		if got := browserOptionDescription(test.value, test.mode); got != test.want {
			t.Errorf("browserOptionDescription(%q, %d) = %q, want %q", test.value, test.mode, got, test.want)
		}
	}
}
