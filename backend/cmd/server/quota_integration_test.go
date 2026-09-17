//go:build integration

package main

import (
	"errors"
	"testing"

	"kisy-backend/internal/groups"
	"kisy-backend/internal/posts"
	"kisy-backend/internal/quota"
)

// Audit A-07: posting and post media were unlimited. Through the real
// communities adapter: posts per hour per account, media per community, and the
// per-file ceiling from configuration.

func TestPostsPerHourAreLimitedPerAccount(t *testing.T) {
	f, seed := newWallFixture(t)
	f.psvc.SetQuota(quota.New(quota.Policy{PostsPerHourBasic: 2, PostsPerHourInvited: 3}))
	basic := seed("poster", 0)
	c, err := f.gsvc.Create(f.ctx, groups.CreateInput{Name: "Own", Kind: groups.KindCommunity, IsPublic: true},
		groups.ActorMeta{UserID: basic.UserID})
	if err != nil {
		t.Fatal(err)
	}
	post := func() error {
		_, err := f.psvc.Create(f.ctx, posts.CreateInput{CommunityID: c.ID, Text: "spam?"}, basic)
		return err
	}
	for i := 0; i < 2; i++ {
		if err := post(); err != nil {
			t.Fatalf("post %d within the hourly limit: %v", i, err)
		}
	}
	if err := post(); !errors.Is(err, quota.ErrPostRate) {
		t.Fatalf("third post in an hour: want ErrPostRate, got %v", err)
	}
}

func TestCommunityMediaQuotaAndPerFileCeiling(t *testing.T) {
	f, seed := newWallFixture(t)
	f.psvc.SetQuota(quota.New(quota.Policy{CommunityBytes: 64 << 10}))
	f.psvc.SetMaxMediaBytes(50 << 10)
	author := seed("author", 5)
	c, err := f.gsvc.Create(f.ctx, groups.CreateInput{Name: "Media", Kind: groups.KindCommunity, IsPublic: true},
		groups.ActorMeta{UserID: author.UserID, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.psvc.Create(f.ctx, posts.CreateInput{CommunityID: c.ID, Text: "gallery"}, author)
	if err != nil {
		t.Fatal(err)
	}
	attach := func(size int) error {
		_, err := f.psvc.AttachMedia(f.ctx, p.ID, posts.UploadedFile{FileName: "x.bin", Bytes: make([]byte, size)}, author)
		return err
	}
	if err := attach(51 << 10); !errors.Is(err, posts.ErrTooLarge) {
		t.Fatalf("media over the configured per-file ceiling: want ErrTooLarge, got %v", err)
	}
	if err := attach(40 << 10); err != nil {
		t.Fatalf("media within the community quota: %v", err)
	}
	if err := attach(40 << 10); !errors.Is(err, quota.ErrCommunityStorage) {
		t.Fatalf("media past the community quota: want ErrCommunityStorage, got %v", err)
	}
}
