package main

import (
	"testing"

	"github.com/slytomcat/yd-go/tools"
	"github.com/stretchr/testify/require"
)

func TestSetupLocalization(t *testing.T) {
	log := tools.SetupLogger(false)
	t.Run("en", func(t *testing.T) {
		t.Setenv("LANG", "en_US.UTF-8")
		p := SetupLocalization(log)
		require.Equal(t, "idle", p.Sprintf("idle"))
	})
	t.Run("ru", func(t *testing.T) {
		t.Setenv("LANG", "ru_RU.UTF-8")
		p := SetupLocalization(log)
		require.Equal(t, "ожидание", p.Sprintf("idle"))
	})
}
