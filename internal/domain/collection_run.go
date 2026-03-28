package domain

import "time"

// CollectionRunOptions configures how a collection is executed.
type CollectionRunOptions struct {
	FolderName    string        // run specific folder only (empty = all)
	Sequential    bool          // true = sequential (default), false = parallel
	StopOnFailure bool          // stop on first failure (default true)
	Delay         time.Duration // delay between requests
	Environment   string        // environment name
	Vars          map[string]string
}

// CollectionRunResult holds the aggregate outcome of running a collection.
type CollectionRunResult struct {
	CollectionName string
	Results        []RequestRunResult
	TotalRequests  int
	Passed         int
	Failed         int
	Skipped        int
	Duration       time.Duration
}

// RequestRunResult holds the outcome of executing a single request in a collection run.
type RequestRunResult struct {
	RequestName string
	FolderPath  string // e.g. "Users/Create User"
	Response    HTTPResponse
	Error       error
	Duration    time.Duration
	Passed      bool
}
