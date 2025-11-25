package mozilla

import (
	"os"
	"testing"
	"time"

	"github.com/blob42/gosuki/internal/utils"
)

func TestCleanOldCopyJobs(t *testing.T) {
	// Setup: create a temporary directory for testing
	tmpDir := t.TempDir()
	utils.TMPDIR = tmpDir

	// Create a copy job with a recent modification time (within the last hour)
	recentJob := NewPlaceCopyJob()
	if _, err := os.Stat(recentJob.Path()); err != nil {
		t.Fatal(err)
	}

	// Create a copy job with an old modification time (more than 1 hour ago)
	oldJob := NewPlaceCopyJob()
	oldInfo, err := os.Stat(oldJob.Path())
	if err != nil {
		t.Fatal(err)
	}

	// Modify the old job's modification time to be more than 1 hour ago
	err = os.Chtimes(oldJob.Path(), oldInfo.ModTime().Add(-2*time.Hour), oldInfo.ModTime().Add(-2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// Ensure both jobs are in the global CopyJobs list
	CopyJobs = []PlaceCopyJob{recentJob, oldJob}

	// Call cleanOldCopyJobs
	err = cleanOldCopyJobs()
	if err != nil {
		t.Fatal(err)
	}

	// Verify that only the recent job remains in CopyJobs
	if len(CopyJobs) != 1 {
		t.Errorf("expected 1 job remaining, got %d", len(CopyJobs))
	}
	if CopyJobs[0].ID != recentJob.ID {
		t.Errorf("expected job with ID %s to remain, got %s", recentJob.ID, CopyJobs[0].ID)
	}

	// Verify that the old job's directory has been removed
	if _, err := os.Stat(oldJob.Path()); !os.IsNotExist(err) {
		t.Errorf("expected old job directory to be removed, but it still exists")
	}
}
