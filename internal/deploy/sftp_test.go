package deploy

import (
	"testing"
	"time"
)

func TestBuildPlanDetectsUploadsUpdatesAndDeletes(t *testing.T) {
	now := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	local := map[string]fileMeta{
		"index.html": {
			rel:     "index.html",
			size:    12,
			modTime: now,
		},
		"posts/a/index.html": {
			rel:     "posts/a/index.html",
			size:    24,
			modTime: now,
		},
		"feed.xml": {
			rel:     "feed.xml",
			size:    8,
			modTime: now,
		},
	}
	remote := map[string]remoteMeta{
		"index.html": {
			rel:     "index.html",
			size:    12,
			modTime: now,
		},
		"posts/a/index.html": {
			rel:     "posts/a/index.html",
			size:    30,
			modTime: now.Add(-time.Hour),
		},
		"old.html": {
			rel:     "old.html",
			size:    5,
			modTime: now,
		},
	}

	plan := buildPlan(local, remote, true)

	if !plan.summary.OutOfSync {
		t.Fatal("OutOfSync = false, want true")
	}
	if plan.summary.Uploaded != 1 || plan.summary.Updated != 1 || plan.summary.Deleted != 1 || plan.summary.Unchanged != 1 {
		t.Fatalf("summary = %#v, want one upload, update, delete, and unchanged", plan.summary)
	}
	if plan.uploads[0].rel != "feed.xml" || plan.updates[0].rel != "posts/a/index.html" || plan.deletes[0].rel != "old.html" {
		t.Fatalf("plan = %#v, want sorted deterministic entries", plan)
	}
}

func TestBuildPlanDoesNotDeleteRemoteOnlyFilesWhenDeleteExtraDisabled(t *testing.T) {
	now := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	plan := buildPlan(
		map[string]fileMeta{
			"index.html": {rel: "index.html", size: 12, modTime: now},
		},
		map[string]remoteMeta{
			"index.html": {rel: "index.html", size: 12, modTime: now},
			"old.html":   {rel: "old.html", size: 5, modTime: now},
		},
		false,
	)

	if plan.summary.OutOfSync {
		t.Fatalf("OutOfSync = true, want false when only remote extras exist and deletion is disabled")
	}
	if plan.summary.RemoteOnly != 1 || plan.summary.Deleted != 0 {
		t.Fatalf("summary = %#v, want remote-only count without deletes", plan.summary)
	}
}
