package storetypes

// IndexStatus describes the state of an index. Is{Valid,Ready,Live} is taken
// from the `pg_index` system table. If the index is currently being created,
// then the remaining reference `fields will be populated describing the index
// creation progress.
type IndexStatus struct {
	IsValid      bool
	IsReady      bool
	IsLive       bool
	Phase        *string
	LockersDone  *int
	LockersTotal *int
	BlocksDone   *int
	BlocksTotal  *int
	TuplesDone   *int
	TuplesTotal  *int
}

// CreateIndexConcurrentlyPhases is an ordered list of phases that occur during
// a CREATE INDEX CONCURRENTLY operation. The phase of an ongoing operation can
// found in the system view `view pg_stat_progress_create_index` (since PG 12).
//
// See https://www.postgresql.org/docs/12/progress-reporting.html#CREATE-INDEX-PROGRESS-REPORTING.
var CreateIndexConcurrentlyPhases = []string{
	"initializing",
	"waiting for writers before build",
	"building index",
	"waiting for writers before validation",
	"index validation: scanning index",
	"index validation: sorting tuples",
	"index validation: scanning table",
	"waiting for old snapshots",
	"waiting for readers before marking dead",
	"waiting for readers before dropping",
}