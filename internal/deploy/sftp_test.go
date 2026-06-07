package deploy

import (
	"context"
	"os"
	"path/filepath"
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

	plan := buildPlan(local, remote)

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

func TestBuildPlanDeletesRemoteOnlyFiles(t *testing.T) {
	now := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	plan := buildPlan(
		map[string]fileMeta{
			"index.html": {rel: "index.html", size: 12, modTime: now},
		},
		map[string]remoteMeta{
			"index.html": {rel: "index.html", size: 12, modTime: now},
			"old.html":   {rel: "old.html", size: 5, modTime: now},
		},
	)

	if !plan.summary.OutOfSync {
		t.Fatalf("OutOfSync = false, want true when remote-only files will be deleted")
	}
	if plan.summary.RemoteOnly != 1 || plan.summary.Deleted != 1 {
		t.Fatalf("summary = %#v, want remote-only count with delete", plan.summary)
	}
}

func TestBuildPlanDetectsLocalModTimeChanges(t *testing.T) {
	now := time.Date(2026, 6, 6, 12, 0, 0, 123, time.UTC)
	plan := buildPlan(
		map[string]fileMeta{
			"index.html": {rel: "index.html", size: 12, modTime: now},
		},
		map[string]remoteMeta{
			"index.html": {rel: "index.html", size: 12, modTime: now.Add(-time.Nanosecond)},
		},
	)

	if plan.summary.Updated != 1 || plan.summary.Unchanged != 0 {
		t.Fatalf("summary = %#v, want same-size local mtime change to update", plan.summary)
	}
}

func TestStatusUsesLocalDeployState(t *testing.T) {
	root := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")
	filePath := filepath.Join(root, "index.html")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat local file: %v", err)
	}
	cfg := Config{
		Host:       "example.com",
		Port:       22,
		User:       "deploy",
		RemotePath: "/public_html",
		StatePath:  statePath,
	}

	summary, err := Status(context.Background(), root, cfg)
	if err != nil {
		t.Fatalf("Status before state returned error: %v", err)
	}
	if !summary.OutOfSync || summary.Uploaded != 1 {
		t.Fatalf("summary before state = %#v, want one local upload", summary)
	}

	if err := saveState(statePath, map[string]remoteMeta{
		"index.html": {
			rel:     "index.html",
			size:    info.Size(),
			modTime: info.ModTime(),
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	summary, err = Status(context.Background(), root, cfg)
	if err != nil {
		t.Fatalf("Status after state returned error: %v", err)
	}
	if summary.OutOfSync || summary.Unchanged != 1 {
		t.Fatalf("summary after state = %#v, want local state to be authoritative", summary)
	}
}

func TestNextStateTracksLocalFilesOnly(t *testing.T) {
	now := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	state := nextState(
		map[string]fileMeta{
			"index.html": {rel: "index.html", size: 12, modTime: now},
		},
	)

	if _, ok := state["old.html"]; ok {
		t.Fatalf("state = %#v, want remote-only file removed from state", state)
	}
	if _, ok := state["index.html"]; !ok {
		t.Fatalf("state = %#v, want local file tracked", state)
	}
}

func TestRemoteFilePathOnlyAllowsPublicRelativePaths(t *testing.T) {
	got, err := remoteFilePath("/home/user/public_html", "posts/a/index.html")
	if err != nil {
		t.Fatalf("remoteFilePath returned error: %v", err)
	}
	if got != "/home/user/public_html/posts/a/index.html" {
		t.Fatalf("remoteFilePath = %q, want path under configured root", got)
	}

	for _, rel := range []string{"/posts/a/index.html", "../index.html", "posts/../index.html", "."} {
		if _, err := remoteFilePath("/home/user/public_html", rel); err == nil {
			t.Fatalf("remoteFilePath(%q) returned nil error, want invalid relative path", rel)
		}
	}
}
