package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateEvalCommand_Flags(t *testing.T) {
	cmd := createEvalCommand()
	assert.Equal(t, "eval", cmd.Name)
	assert.Contains(t, cmd.Usage, "quality")

	flagNames := map[string]bool{}
	for _, f := range cmd.Flags {
		flagNames[f.Names()[0]] = true
	}
	assert.True(t, flagNames["repo"])
	assert.True(t, flagNames["fixtures"])
	assert.True(t, flagNames["db"])
	assert.True(t, flagNames["only"])
	assert.True(t, flagNames["format"])
}

func TestCreateEvalCommand_HasAction(t *testing.T) {
	cmd := createEvalCommand()
	assert.NotNil(t, cmd.Action)
}
