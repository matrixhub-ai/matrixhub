package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/cleanup"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

type fakeCleanupService struct {
	calls      int
	opts       cleanup.CleanupOptions
	result     *cleanup.CleanupResult
	err        error
	statsCalls int
	stats      *cleanup.StorageStats
	statsErr   error
}

func (f *fakeCleanupService) ExecuteCleanup(_ context.Context, opts cleanup.CleanupOptions) (*cleanup.CleanupResult, error) {
	f.calls++
	f.opts = opts
	return f.result, f.err
}

func (f *fakeCleanupService) GetStorageStats(context.Context) (*cleanup.StorageStats, error) {
	f.statsCalls++
	return f.stats, f.statsErr
}

func TestCleanupHandlerExecuteForwardsOptions(t *testing.T) {
	hour, disabled := time.Hour, time.Duration(-1)
	fromJSON := func(s string) *v1alpha1.ExecuteCleanupRequest {
		req := &v1alpha1.ExecuteCleanupRequest{}
		if err := protojson.Unmarshal([]byte(s), req); err != nil {
			t.Fatalf("protojson.Unmarshal(%s) error = %v", s, err)
		}
		return req
	}
	cases := []struct {
		name string
		req  *v1alpha1.ExecuteCleanupRequest
		want cleanup.CleanupOptions
	}{
		{"grace absent keeps the configured grace", &v1alpha1.ExecuteCleanupRequest{CleanOrphanedRepos: true, CleanOrphanedLfs: true, DryRun: true},
			cleanup.CleanupOptions{CleanOrphanedRepos: true, CleanOrphanedLFS: true, PruneOptions: git.PruneOptions{DryRun: true}}},
		{"explicit zero grace disables it", &v1alpha1.ExecuteCleanupRequest{CleanOrphanedLfs: true, Grace: durationpb.New(0)},
			cleanup.CleanupOptions{CleanOrphanedLFS: true, PruneOptions: git.PruneOptions{Grace: &disabled}}},
		{"positive grace and bounds forwarded", &v1alpha1.ExecuteCleanupRequest{CleanOrphanedLfs: true, Grace: durationpb.New(time.Hour), MaxDeletes: 7, Budget: durationpb.New(30 * time.Second)},
			cleanup.CleanupOptions{CleanOrphanedLFS: true, PruneOptions: git.PruneOptions{Grace: &hour, MaxDeletes: 7, Budget: 30 * time.Second}}},
		{"JSON wire format", fromJSON(`{"cleanOrphanedLfs":true,"grace":"0s","maxDeletes":1,"budget":"1.5s"}`),
			cleanup.CleanupOptions{CleanOrphanedLFS: true, PruneOptions: git.PruneOptions{Grace: &disabled, MaxDeletes: 1, Budget: 1500 * time.Millisecond}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeCleanupService{result: &cleanup.CleanupResult{}}
			h := NewCleanupHandler(svc).(*CleanupHandler)
			if _, err := h.ExecuteCleanup(t.Context(), tc.req); err != nil {
				t.Fatalf("ExecuteCleanup() error = %v", err)
			}
			if svc.calls != 1 {
				t.Fatalf("service calls = %d, want 1", svc.calls)
			}
			got := svc.opts
			if (got.Grace == nil) != (tc.want.Grace == nil) || (got.Grace != nil && *got.Grace != *tc.want.Grace) {
				t.Fatalf("Grace = %v, want %v", got.Grace, tc.want.Grace)
			}
			got.Grace, tc.want.Grace = nil, nil
			if got != tc.want {
				t.Fatalf("options = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestCleanupHandlerExecuteRejectsNegativeValues(t *testing.T) {
	cases := []struct {
		name string
		req  *v1alpha1.ExecuteCleanupRequest
	}{
		{"grace", &v1alpha1.ExecuteCleanupRequest{Grace: durationpb.New(-time.Second)}},
		{"max deletes", &v1alpha1.ExecuteCleanupRequest{MaxDeletes: -1}},
		{"budget", &v1alpha1.ExecuteCleanupRequest{Budget: durationpb.New(-time.Second)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeCleanupService{result: &cleanup.CleanupResult{}}
			h := NewCleanupHandler(svc).(*CleanupHandler)
			res, err := h.ExecuteCleanup(t.Context(), tc.req)
			if status.Code(err) != codes.InvalidArgument || res != nil {
				t.Fatalf("ExecuteCleanup() = %v, %v; want InvalidArgument", res, err)
			}
			if svc.calls != 0 {
				t.Fatalf("service calls = %d, want 0", svc.calls)
			}
		})
	}
}

func TestCleanupHandlerExecuteMapsResult(t *testing.T) {
	done, unfinished := true, false
	full := &cleanup.CleanupResult{
		OrphanedRepos:  []string{"p/a.git", "p/b.git"},
		SpaceReclaimed: 101,
		Errors:         []string{"walk failed"},
		GC: &git.GCResult{DryRun: true, Repositories: 2, DeletedGitObjects: 3, DeletedGitBytes: 4, GitReclaimedBytes: 5,
			Failed: map[string]string{"p/failed.git": "gc failed"}, LiveObjects: 6, Unlinked: []string{"aa", "bb"}, PruneSkippedInGrace: 7,
			SweptShards: 8, SweptXorbs: 9, XetReclaimedBytes: 10, SweepSkippedInGrace: 11, Dangling: []string{"d1"}, UnreadableShards: []string{"u1"},
			SweepDone: &done, RemainingShards: 12, RemainingXorbs: 13},
	}
	wantFull := &v1alpha1.CleanupResult{
		SpaceReclaimedBytes: 101,
		Errors:              []string{"walk failed"},
		OrphanedRepos:       []string{"p/a.git", "p/b.git"},
		Gc: &v1alpha1.GCResult{DryRun: true, Repositories: 2, DeletedGitObjects: 3, DeletedGitBytes: 4, GitReclaimedBytes: 5,
			Failed: map[string]string{"p/failed.git": "gc failed"}, LiveObjects: 6, Unlinked: []string{"aa", "bb"}, PruneSkippedInGrace: 7,
			SweptShards: 8, SweptXorbs: 9, XetReclaimedBytes: 10, SweepSkippedInGrace: 11, Dangling: []string{"d1"}, UnreadableShards: []string{"u1"},
			SweepDone: wrapperspb.Bool(true), RemainingShards: 12, RemainingXorbs: 13},
	}
	cases := []struct {
		name   string
		result *cleanup.CleanupResult
		want   *v1alpha1.CleanupResult
	}{
		{"all fields distinct", full, wantFull},
		{"no gc section", &cleanup.CleanupResult{OrphanedRepos: []string{"p/a.git"}, SpaceReclaimed: 5}, &v1alpha1.CleanupResult{OrphanedRepos: []string{"p/a.git"}, SpaceReclaimedBytes: 5}},
		{"partial gc without a sweep result", &cleanup.CleanupResult{GC: &git.GCResult{Unlinked: []string{"aa"}, GitReclaimedBytes: 9}, Errors: []string{"sweep failed"}},
			&v1alpha1.CleanupResult{Gc: &v1alpha1.GCResult{Unlinked: []string{"aa"}, GitReclaimedBytes: 9}, Errors: []string{"sweep failed"}}},
		{"unfinished bounded sweep", &cleanup.CleanupResult{GC: &git.GCResult{SweepDone: &unfinished, RemainingShards: 1}},
			&v1alpha1.CleanupResult{Gc: &v1alpha1.GCResult{SweepDone: wrapperspb.Bool(false), RemainingShards: 1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewCleanupHandler(&fakeCleanupService{result: tc.result}).(*CleanupHandler)
			got, err := h.ExecuteCleanup(t.Context(), &v1alpha1.ExecuteCleanupRequest{CleanOrphanedRepos: true, CleanOrphanedLfs: true})
			if err != nil {
				t.Fatalf("ExecuteCleanup() error = %v", err)
			}
			if !proto.Equal(got, tc.want) {
				t.Fatalf("ExecuteCleanup() = %v, want %v", got, tc.want)
			}
		})
	}
}

// With EmitUnpopulated a missing sweep result is null while false stays explicit.
func TestCleanupHandlerExecuteSweepDoneTriState(t *testing.T) {
	done, unfinished := true, false
	cases := []struct {
		name      string
		sweepDone *bool
		want      string
	}{
		{"absent", nil, "null"},
		{"false", &unfinished, "false"},
		{"true", &done, "true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewCleanupHandler(&fakeCleanupService{result: &cleanup.CleanupResult{GC: &git.GCResult{SweepDone: tc.sweepDone, RemainingShards: 1}}}).(*CleanupHandler)
			result, err := h.ExecuteCleanup(t.Context(), &v1alpha1.ExecuteCleanupRequest{CleanOrphanedLfs: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, emitUnpopulated := range []bool{false, true} {
				encoded, err := protojson.MarshalOptions{EmitUnpopulated: emitUnpopulated}.Marshal(result)
				if err != nil {
					t.Fatal(err)
				}
				var fields struct{ Gc map[string]json.RawMessage }
				if err := json.Unmarshal(encoded, &fields); err != nil {
					t.Fatal(err)
				}
				got, present := fields.Gc["sweepDone"]
				if wantPresent := emitUnpopulated || tc.sweepDone != nil; present != wantPresent || (present && string(got) != tc.want) {
					t.Fatalf("EmitUnpopulated=%t: %s, want sweepDone %s (present=%t)", emitUnpopulated, encoded, tc.want, wantPresent)
				}
				decoded := &v1alpha1.CleanupResult{}
				if err := protojson.Unmarshal(encoded, decoded); err != nil || !proto.Equal(decoded, result) {
					t.Fatalf("protojson roundtrip = %v, %v; want %v", decoded, err, result)
				}
			}
		})
	}
}

func TestCleanupHandlerExecuteResultJSON(t *testing.T) {
	handler := NewCleanupHandler(&fakeCleanupService{result: &cleanup.CleanupResult{
		GC: &git.GCResult{DeletedGitObjects: 2, SweptShards: 3},
	}}).(*CleanupHandler)
	result, err := handler.ExecuteCleanup(t.Context(), &v1alpha1.ExecuteCleanupRequest{CleanOrphanedLfs: true})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := protojson.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 || fields["gc"] == nil {
		t.Fatalf("cleanup result JSON = %s, want one gc section", encoded)
	}
	descriptor := result.ProtoReflect().Descriptor()
	gc := descriptor.Fields().ByName("gc")
	if gc == nil || gc.Number() != 8 {
		t.Fatal("gc must use a new protobuf field number")
	}
	if !descriptor.ReservedRanges().Has(6) || !descriptor.ReservedRanges().Has(7) {
		t.Fatal("old prune and sweep field numbers must remain reserved")
	}
	if !descriptor.ReservedNames().Has("prune") || !descriptor.ReservedNames().Has("sweep") ||
		!descriptor.ReservedNames().Has("lfs_prune") || !descriptor.ReservedNames().Has("lfs_sweep") {
		t.Fatal("old cleanup result field names must remain reserved")
	}
}

func TestCleanupHandlerExecuteReturnsServiceError(t *testing.T) {
	want := errors.New("gc busy")
	h := NewCleanupHandler(&fakeCleanupService{err: want}).(*CleanupHandler)
	res, err := h.ExecuteCleanup(t.Context(), &v1alpha1.ExecuteCleanupRequest{CleanOrphanedLfs: true})
	if !errors.Is(err, want) || res != nil {
		t.Fatalf("ExecuteCleanup() = %v, %v; want %v", res, err, want)
	}
}

func TestCleanupHandlerGetStorageStatsMapsEveryCategory(t *testing.T) {
	svc := &fakeCleanupService{stats: &cleanup.StorageStats{
		TotalSizeBytes: 280,
		Git:            git.GitUsage{Objects: git.ObjectUsage{Count: 1, Bytes: 10}, Other: git.ObjectUsage{Count: 2, Bytes: 20}},
		Xet: git.XetUsage{
			Xorbs:       git.ObjectUsage{Count: 3, Bytes: 30},
			Shards:      git.ObjectUsage{Count: 4, Bytes: 40},
			FileIndex:   git.ObjectUsage{Count: 5, Bytes: 50},
			ChunkIndex:  git.ObjectUsage{Count: 6, Bytes: 60},
			SHA256Index: git.ObjectUsage{Count: 7, Bytes: 70},
		},
	}}
	want := &v1alpha1.StorageStats{
		TotalSizeBytes: 280,
		Git: &v1alpha1.GitStorageUsage{
			Objects: &v1alpha1.StorageObjectUsage{Count: 1, Bytes: 10},
			Other:   &v1alpha1.StorageObjectUsage{Count: 2, Bytes: 20},
		},
		Xet: &v1alpha1.XetStorageUsage{
			Xorbs:       &v1alpha1.StorageObjectUsage{Count: 3, Bytes: 30},
			Shards:      &v1alpha1.StorageObjectUsage{Count: 4, Bytes: 40},
			FileIndex:   &v1alpha1.StorageObjectUsage{Count: 5, Bytes: 50},
			ChunkIndex:  &v1alpha1.StorageObjectUsage{Count: 6, Bytes: 60},
			Sha256Index: &v1alpha1.StorageObjectUsage{Count: 7, Bytes: 70},
		},
	}
	h := NewCleanupHandler(svc).(*CleanupHandler)
	got, err := h.GetStorageStats(t.Context(), &v1alpha1.GetStorageStatsRequest{})
	if err != nil {
		t.Fatalf("GetStorageStats() error = %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("GetStorageStats() = %v, want %v", got, want)
	}
	if svc.statsCalls != 1 || svc.calls != 0 {
		t.Fatalf("service calls: stats = %d, cleanup = %d; want 1 and 0", svc.statsCalls, svc.calls)
	}
}

func TestCleanupHandlerGetStorageStatsReturnsServiceError(t *testing.T) {
	want := errors.New("xet usage: store offline")
	h := NewCleanupHandler(&fakeCleanupService{statsErr: want}).(*CleanupHandler)
	res, err := h.GetStorageStats(t.Context(), &v1alpha1.GetStorageStatsRequest{})
	if !errors.Is(err, want) || res != nil {
		t.Fatalf("GetStorageStats() = %v, %v; want %v", res, err, want)
	}
}
