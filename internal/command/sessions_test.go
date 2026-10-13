package command

import "testing"

func TestSessionsCommandIsRegistered(t *testing.T) {
	info, ok := GetCommandRegistry().Commands["/sessions"]
	if !ok {
		t.Fatal("/sessions is not registered")
	}
	if info.Usage != "/sessions [query]" || info.Category != "core" {
		t.Fatalf("/sessions registration = %+v", info)
	}
	if len(info.Examples) == 0 {
		t.Fatal("/sessions has no help example")
	}
}
