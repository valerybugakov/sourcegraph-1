package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/go-ctags"
	"golang.org/x/sync/semaphore"

	"github.com/sourcegraph/sourcegraph/cmd/symbols/fetcher"
	symbolsGitserver "github.com/sourcegraph/sourcegraph/cmd/symbols/gitserver"
	symbolsParser "github.com/sourcegraph/sourcegraph/cmd/symbols/parser"
	"github.com/sourcegraph/sourcegraph/cmd/symbols/shared"
	"github.com/sourcegraph/sourcegraph/cmd/symbols/types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/rockskip"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/conf/conftypes"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	connections "github.com/sourcegraph/sourcegraph/internal/database/connections/live"
	"github.com/sourcegraph/sourcegraph/internal/env"
	gitserver "github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
)

func main() {
	if env.Get("USE_ROCKSKIP", "false", "use Rockskip instead of SQLite") == "true" {
		shared.Main(SetupRockskip)
	} else {
		shared.Main(shared.SetupSqlite)
	}
}

func SetupRockskip(observationContext *observation.Context) (types.SearchFunc, func(http.ResponseWriter, *http.Request), []goroutine.BackgroundRoutine) {
	baseConfig := env.BaseConfig{}
	config := LoadRockskipConfig(baseConfig)
	if err := baseConfig.Validate(); err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	db := mustInitializeCodeIntelDB()

	statuses := NewStatuses()
	searchFunc, err := MakeRockskipSearchFunc(observationContext, db, config, statuses)
	if err != nil {
		log.Fatalf("Failed to create rockskip search function: %s", err)
	}

	return searchFunc, handleStatus(db, statuses), nil
}

// RequestId is a unique int for each HTTP request.
type RequestId = int

// Status contains the status of all requests.
type Status struct {
	requestToStatus map[RequestId]*rockskip.Status
	nextRequestId   RequestId
	mu              sync.Mutex
}

func NewStatuses() Status {
	return Status{
		requestToStatus: map[int]*rockskip.Status{},
		nextRequestId:   0,
		mu:              sync.Mutex{},
	}
}

func (status *Status) Begin(repo string, commit string) (*rockskip.Status, func()) {
	status.mu.Lock()
	defer status.mu.Unlock()

	rockskipStatus := rockskip.NewStatus(repo, commit)

	requestId := status.nextRequestId
	status.nextRequestId++
	status.requestToStatus[requestId] = rockskipStatus

	return rockskipStatus, func() {
		status.mu.Lock()
		defer status.mu.Unlock()
		delete(status.requestToStatus, requestId)
	}
}

func handleStatus(db *sql.DB, statuses Status) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		repositoryCount, _, err := basestore.ScanFirstInt(db.QueryContext(ctx, "SELECT COUNT(*) FROM rockskip_repos"))
		if err != nil {
			log15.Error("Failed to count repos", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		blobCount, _, err := basestore.ScanFirstInt(db.QueryContext(ctx, "SELECT COUNT(*) FROM rockskip_blobs"))
		if err != nil {
			log15.Error("Failed to count blobs", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		blobsSize, _, err := basestore.ScanFirstString(db.QueryContext(ctx, "SELECT pg_size_pretty(pg_total_relation_size('rockskip_blobs'))"))
		if err != nil {
			log15.Error("Failed to get size of blobs table", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "This is the symbols service status page.")
		fmt.Fprintln(w, "")

		fmt.Fprintf(w, "Number of repositories: %d\n", repositoryCount)
		fmt.Fprintf(w, "Number of blobs indexed: %d\n", blobCount)
		fmt.Fprintf(w, "Size of blobs table: %s\n", blobsSize)
		fmt.Fprintln(w, "")

		statuses.mu.Lock()
		defer statuses.mu.Unlock()

		if len(statuses.requestToStatus) == 0 {
			fmt.Fprintln(w, "No requests in flight.")
			return
		}
		fmt.Fprintln(w, "Here's all in-flight requests:")
		fmt.Fprintln(w, "")

		ids := []int{}
		for id := range statuses.requestToStatus {
			ids = append(ids, id)
		}
		sort.Ints(ids)

		for _, id := range ids {
			status := statuses.requestToStatus[id]

			fmt.Fprintf(w, "%s@%s\n", status.Repo, status.Commit)
			if status.Total != 0 {
				fmt.Fprintf(w, "    progress %.2f%% (indexed %d of %d commits)\n", float64(status.Indexed)/float64(status.Total)*100, status.Indexed, status.Total)
			}
			fmt.Fprintf(w, "    %s\n", status.TaskLog)
			blockedOn := status.BlockedOn
			if blockedOn != "" {
				fmt.Fprintf(w, "    blocked on %s\n", blockedOn)
			}
			for name := range status.HeldLocks {
				fmt.Fprintf(w, "    holding %s\n", name)
			}
			fmt.Fprintln(w)
		}
	}
}

type RockskipConfig struct {
	Ctags                   types.CtagsConfig
	RepositoryFetcher       types.RepositoryFetcherConfig
	MaxRepos                int
	MaxConcurrentlyIndexing int
}

func LoadRockskipConfig(baseConfig env.BaseConfig) RockskipConfig {
	return RockskipConfig{
		Ctags:                   types.LoadCtagsConfig(baseConfig),
		RepositoryFetcher:       types.LoadRepositoryFetcherConfig(baseConfig),
		MaxRepos:                baseConfig.GetInt("MAX_REPOS", "1000", "maximum number of repositories for Rockskip to store in Postgres, with LRU eviction"),
		MaxConcurrentlyIndexing: baseConfig.GetInt("MAX_CONCURRENTLY_INDEXING", "10", "maximum number of repositories to index at a time"),
	}
}

func MakeRockskipSearchFunc(observationContext *observation.Context, db *sql.DB, config RockskipConfig, status Status) (types.SearchFunc, error) {
	gitserverClient := symbolsGitserver.NewClient(observationContext)

	f := fetcher.NewRepositoryFetcher(gitserverClient, config.RepositoryFetcher.MaxTotalPathsLength, observationContext)

	sem := semaphore.NewWeighted(int64(config.MaxConcurrentlyIndexing))

	return func(ctx context.Context, args types.SearchArgs) (results *[]result.Symbol, err error) {
		fmt.Println("🔵 Rockskip search", args.Repo, args.CommitID, args.Query)
		defer func() {
			if results == nil {
				fmt.Println("🔴 Rockskip search failed")
			} else {
				for _, result := range *results {
					fmt.Println("  -", result.Path+":"+fmt.Sprint(result.Line), result.Name)
				}
				fmt.Println("🔴 Rockskip search", len(*results))
			}
		}()

		// Lazily create the parser
		var parser ctags.Parser
		createParserOnce := sync.Once{}
		defer func() {
			if parser != nil {
				parser.Close()
			}
		}()

		var parse rockskip.ParseSymbolsFunc = func(path string, bytes []byte) (symbols []rockskip.Symbol, err error) {
			createParserOnce.Do(func() {
				parser = mustCreateCtagsParser(config.Ctags)
			})
			entries, err := parser.Parse(path, bytes)
			if err != nil {
				return nil, err
			}

			symbols = []rockskip.Symbol{}
			for _, entry := range entries {
				symbols = append(symbols, rockskip.Symbol{
					Name:   entry.Name,
					Parent: entry.Parent,
					Kind:   entry.Kind,
					Line:   entry.Line,
				})
			}

			return symbols, nil
		}

		rockskipStatus, end := status.Begin(string(args.Repo), string(args.CommitID))
		err = rockskip.Index(NewGitserver(f, string(args.Repo)), db, rockskipStatus, parse, string(args.Repo), string(args.CommitID), config.MaxRepos, sem)
		end()
		if err != nil {
			return nil, errors.Wrap(err, "rockskip.Index")
		}

		var query *string
		if args.Query != "" {
			query = &args.Query
		}
		tasklog2 := rockskip.NewTaskLog()
		blobs, err := rockskip.Search(db, tasklog2, string(args.Repo), string(args.CommitID), query)
		tasklog2.Print()
		if err != nil {
			return nil, errors.Wrap(err, "rockskip.Search")
		}

		res := []result.Symbol{}
		for _, blob := range blobs {
			for _, symbol := range blob.Symbols {
				res = append(res, result.Symbol{
					Name:   symbol.Name,
					Path:   blob.Path,
					Line:   symbol.Line,
					Kind:   symbol.Kind,
					Parent: symbol.Parent,
				})
			}
		}

		return &res, err
	}, nil
}

func mustInitializeCodeIntelDB() *sql.DB {
	dsn := conf.GetServiceConnectionValueAndRestartOnChange(func(serviceConnections conftypes.ServiceConnections) string {
		return serviceConnections.CodeIntelPostgresDSN
	})
	var (
		db  *sql.DB
		err error
	)
	if os.Getenv("NEW_MIGRATIONS") == "" {
		// CURRENTLY DEPRECATING
		db, err = connections.NewCodeIntelDB(dsn, "symbols", true, &observation.TestContext)
	} else {
		db, err = connections.EnsureNewCodeIntelDB(dsn, "symbols", &observation.TestContext)
	}
	if err != nil {
		log.Fatalf("Failed to connect to codeintel database: %s", err)
	}

	return db
}

func mustCreateCtagsParser(ctagsConfig types.CtagsConfig) ctags.Parser {
	options := ctags.Options{
		Bin:                ctagsConfig.Command,
		PatternLengthLimit: ctagsConfig.PatternLengthLimit,
	}
	if ctagsConfig.LogErrors {
		options.Info = log.New(os.Stderr, "ctags: ", log.LstdFlags)
	}
	if ctagsConfig.DebugLogs {
		options.Debug = log.New(os.Stderr, "DBUG ctags: ", log.LstdFlags)
	}

	parser, err := ctags.New(options)
	if err != nil {
		log.Fatalf("Failed to create new ctags parser: %s", err)
	}

	return symbolsParser.NewFilteringParser(parser, ctagsConfig.MaxFileSize, ctagsConfig.MaxSymbols)
}

type Gitserver struct {
	repositoryFetcher fetcher.RepositoryFetcher
	repo              string
}

func NewGitserver(repositoryFetcher fetcher.RepositoryFetcher, repo string) rockskip.Git {
	return Gitserver{
		repositoryFetcher: repositoryFetcher,
		repo:              repo,
	}
}

func (g Gitserver) LogReverseEach(commit string, n int, onLogEntry func(entry rockskip.LogEntry) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	command := gitserver.DefaultClient.Command("git", rockskip.LogReverseArgs(n, commit)...)
	command.Repo = api.RepoName(g.repo)
	// We run a single `git log` command and stream the output while the repo is being processed, which
	// can take much longer than 1 minute (the default timeout).
	command.DisableTimeout()
	stdout, err := gitserver.StdoutReader(ctx, command)
	if err != nil {
		return err
	}
	defer stdout.Close()

	return errors.Wrap(rockskip.ParseLogReverseEach(stdout, onLogEntry), "ParseLogReverseEach")
}

func (g Gitserver) RevListEach(commit string, onCommit func(commit string) (shouldContinue bool, err error)) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	command := gitserver.DefaultClient.Command("git", rockskip.RevListArgs(commit)...)
	command.Repo = api.RepoName(g.repo)
	command.DisableTimeout()
	stdout, err := gitserver.StdoutReader(ctx, command)
	if err != nil {
		return err
	}
	defer stdout.Close()

	return rockskip.RevListEach(stdout, onCommit)
}

func (g Gitserver) ArchiveEach(commit string, paths []string, onFile func(path string, contents []byte) error) error {
	if len(paths) == 0 {
		return nil
	}

	args := types.SearchArgs{Repo: api.RepoName(g.repo), CommitID: api.CommitID(commit)}
	parseRequestOrErrors := g.repositoryFetcher.FetchRepositoryArchive(context.TODO(), args, paths)
	defer func() {
		// Ensure the channel is drained
		for range parseRequestOrErrors {
		}
	}()

	for parseRequestOrError := range parseRequestOrErrors {
		if parseRequestOrError.Err != nil {
			return errors.Wrap(parseRequestOrError.Err, "FetchRepositoryArchive")
		}

		err := onFile(parseRequestOrError.ParseRequest.Path, parseRequestOrError.ParseRequest.Data)
		if err != nil {
			return err
		}
	}

	return nil
}
