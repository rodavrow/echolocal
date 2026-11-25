//go:build !board_biscuit

package all

import (
	"testing"

	"github.com/ygelfand/echolocal/internal/component"
)

// The Dot has no screen to caption, so biscuit_test.go's list holds it off that board.
func TestScreenBoardsHaveTheShowCaptionsSwitch(t *testing.T) {
	for _, e := range component.Default().Entities() {
		if objectID(e) == "assistants_show_captions" {
			return
		}
	}
	t.Error("no entity has the object id assistants_show_captions")
}
