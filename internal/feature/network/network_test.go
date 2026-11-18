package network

import (
	"slices"
	"testing"
)

func TestOnChangeAppliesTheFirstValueWhenItDiffersFromTheStart(t *testing.T) {
	var applied []bool
	keep := onChange(false, func(on bool) { applied = append(applied, on) })

	keep(true)

	if !slices.Equal(applied, []bool{true}) {
		t.Errorf("applied %v, want the new value once", applied)
	}
}

func TestOnChangeSkipsAValueThatDidNotChange(t *testing.T) {
	var applied []bool
	keep := onChange(true, func(on bool) { applied = append(applied, on) })

	keep(true)
	keep(true)

	if len(applied) != 0 {
		t.Errorf("applied %v for changes that left the value alone", applied)
	}
}

func TestOnChangeAppliesEachFlip(t *testing.T) {
	var applied []bool
	keep := onChange(false, func(on bool) { applied = append(applied, on) })

	keep(true)
	keep(true)
	keep(false)

	if !slices.Equal(applied, []bool{true, false}) {
		t.Errorf("applied %v, want on then off", applied)
	}
}
