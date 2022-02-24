package repos

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/sourcegraph/sourcegraph/internal/api"
	codeinteldbstore "github.com/sourcegraph/sourcegraph/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/lockfiles"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type DependenciesResolver interface {
	Dependencies(ctx context.Context, repoRevs map[string][]string) (_ map[string][]string, err error)
}

type dependenciesResolver struct {
	db       database.DB
	listFunc func(context.Context, database.ExternalServicesListOptions) (dependencyRevs []*types.ExternalService, err error)
	syncFunc func(context.Context, int64) error
}

func NewDependenciesResolver(
	db database.DB,
	listFunc func(context.Context, database.ExternalServicesListOptions) (dependencyRevs []*types.ExternalService, err error),
	syncFunc func(context.Context, int64) error,
) *dependenciesResolver {
	return &dependenciesResolver{
		db:       db,
		listFunc: listFunc,
		syncFunc: syncFunc,
	}
}

// Dependencies resolves the (transitive) dependencies for a set of repository and revisions.
// Both the input repoRevs and the output dependencyRevs are a map from repository names to
// `40-char commit hashes belonging to that repository.
func (r *dependenciesResolver) Dependencies(ctx context.Context, repoRevs map[string][]string) (_ map[string][]string, err error) {
	depsStore := codeinteldbstore.NewDependencyInserter(r.db, r.listFunc, r.syncFunc)
	defer func() {
		if flushErr := depsStore.Flush(context.Background()); flushErr != nil {
			err = errors.Append(err, flushErr)
		}
	}()

	var (
		mu             sync.Mutex
		dependencyRevs = make(map[string]map[string]struct{})
	)

	rg, ctx := errgroup.WithContext(ctx)
	svc := &lockfiles.Service{GitArchive: gitserver.DefaultClient.Archive}
	sem := semaphore.NewWeighted(16)

	for repoName, revs := range repoRevs {
		for _, rev := range revs {
			repoName, rev := repoName, rev

			rg.Go(func() error {
				if err := sem.Acquire(ctx, 1); err != nil {
					return err
				}
				defer sem.Release(1)

				return svc.StreamDependencies(ctx, api.RepoName(repoName), rev, func(dep reposource.PackageDependency) error {
					if err := depsStore.Insert(ctx, dep); err != nil {
						return err
					}

					depName := string(dep.RepoName())
					depRev := dep.GitTagFromVersion()

					mu.Lock()
					defer mu.Unlock()

					if _, ok := dependencyRevs[depName]; !ok {
						dependencyRevs[depName] = map[string]struct{}{}
					}

					dependencyRevs[depName][depRev] = struct{}{}
					return nil
				})
			})
		}
	}
	if err := rg.Wait(); err != nil {
		return nil, err
	}

	flattenedDependencyRevs := make(map[string][]string, len(dependencyRevs))
	for k, vs := range dependencyRevs {
		keys := make([]string, 0, len(vs))
		for v := range vs {
			keys = append(keys, v)
		}

		flattenedDependencyRevs[k] = keys
	}

	return flattenedDependencyRevs, nil
}
