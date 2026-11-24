//go:build board_biscuit

// Package avatar has an assistant in avatar mode shown and voiced by its LiveTalking avatar.
// Biscuit has no screen, so it has no avatars.
package avatar

import "github.com/ygelfand/libcountertop/pkg/assistant/turn"

func Turn() turn.Avatar { return nil }

func Chosen(int, bool) {}
