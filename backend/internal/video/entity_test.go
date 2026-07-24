package video

import (
	"testing"
	"time"
)

func TestStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to Status
		want     bool
	}{
		{StatusDraft, StatusPublished, true}, {StatusPublished, StatusHidden, true}, {StatusHidden, StatusPublished, true},
		{StatusDraft, StatusHidden, false}, {StatusDeleted, StatusPublished, false},
	}
	for _, tc := range cases {
		if got := canTransition(tc.from, tc.to); got != tc.want {
			t.Errorf("canTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestValidatePublish(t *testing.T) {
	item := &Video{Title: "short video", PlayURL: "https://cdn.example/video.mp4", CoverURL: "https://cdn.example/cover.jpg"}
	if err := validatePublish(item); err != nil {
		t.Fatalf("valid video rejected: %v", err)
	}
	item.PlayURL = "file:///video.mp4"
	if err := validatePublish(item); err == nil {
		t.Fatal("invalid media URL accepted")
	}
}

func TestHotWindowKeyUsesUTCMinute(t *testing.T) {
	value := hotWindowKey(time.Date(2026, 7, 24, 12, 34, 56, 0, time.FixedZone("test", 8*60*60)))
	if value != "feed:hot:window:202607240434" {
		t.Fatalf("unexpected key: %s", value)
	}
}

func TestEncodeCursorRoundTrip(t *testing.T) {
	want := cursor{SnapshotID: "snapshot-1", HotOffset: 10, ColdCursor: "2026-07-24T04:34:00Z"}
	encoded := EncodeCursor(want)
	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != want {
		t.Fatalf("decoded cursor %#v, want %#v", decoded, want)
	}
}
