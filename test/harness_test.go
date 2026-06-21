package test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/blob42/gosuki/test/fixtures"
)

func TestHarnessSmoke(t *testing.T) {
	h := NewHarness(t)
	defer h.Cleanup()

	// Verify DB is accessible
	require.NotNil(t, h.DB)
	require.NotEmpty(t, h.DBPath)
	require.NotEmpty(t, h.Dir)

	// Seed and verify
	count := h.SeedAndCount(fixtures.DefaultSeedSet())
	require.Equal(t, 5, count)
}

func TestHarnessWithSeed(t *testing.T) {
	h := NewHarnessWithSeed(t, fixtures.DefaultSeedSet())
	defer h.Cleanup()

	var count int
	err := h.DB.Handle.Get(&count, "SELECT COUNT(*) FROM gskbookmarks")
	require.NoError(t, err)
	require.Equal(t, 5, count)
}

func TestRunWithDB(t *testing.T) {
	RunWithDB(t, func(h *Harness) {
		count := h.SeedAndCount(fixtures.TagVarietySet())
		require.Equal(t, 5, count)
	})
}
