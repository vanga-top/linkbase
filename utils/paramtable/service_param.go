package paramtable

// --- rocksmq ---
type RocksmqConfig struct {
	Path          ParamItem `refreshable:"false"`
	LRUCacheRatio ParamItem `refreshable:"false"`
	PageSize      ParamItem `refreshable:"false"`
	// RetentionTimeInMinutes is the time of retention
	RetentionTimeInMinutes ParamItem `refreshable:"false"`
	// RetentionSizeInMB is the size of retention
	RetentionSizeInMB ParamItem `refreshable:"false"`
	// CompactionInterval is the Interval we trigger compaction,
	CompactionInterval ParamItem `refreshable:"false"`
	// TickerTimeInSeconds is the time of expired check, default 10 minutes
	TickerTimeInSeconds ParamItem `refreshable:"false"`
	// CompressionTypes is compression type of each level
	// len of CompressionTypes means num of rocksdb level.
	// only support {0,7}, 0 means no compress, 7 means zstd
	// default [0,7].
	CompressionTypes ParamItem `refreshable:"false"`
}
