package lockfiles

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type Service struct {
	GitArchive func(context.Context, api.RepoName, gitserver.ArchiveOptions) (io.ReadCloser, error)
}

func (s *Service) StreamDependencies(ctx context.Context, repo api.RepoName, rev string, cb func(reposource.PackageDependency) error) (err error) {
	tr, ctx := trace.New(ctx, "lockfiles.StreamDependencies", string(repo)+"@"+rev)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	opts := gitserver.ArchiveOptions{
		Treeish: rev,
		Format:  "zip",
		Paths: []string{
			"*" + NPMFilename,
		},
	}

	rc, err := s.GitArchive(ctx, repo, opts)
	if err != nil {
		return err
	}

	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		if strings.Contains(err.Error(), "did not match any files") {
			return nil
		}
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	set := map[string]struct{}{}
	for _, f := range zr.File {
		if f.Mode().IsDir() {
			continue
		}

		ds, err := parseZipLockfile(f)
		if err != nil {
			return errors.Wrapf(err, "failed to parse %q", f.Name)
		}

		for _, d := range ds {
			k := d.PackageManagerSyntax()
			if _, ok := set[k]; !ok {
				set[k] = struct{}{}
				if err = cb(d); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *Service) ListDependencies(ctx context.Context, repo api.RepoName, rev string) (deps []reposource.PackageDependency, err error) {
	tr, ctx := trace.New(ctx, "lockfiles.ListDependencies", string(repo)+"@"+rev)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	err = s.StreamDependencies(ctx, repo, rev, func(d reposource.PackageDependency) error {
		deps = append(deps, d)
		return nil
	})

	return deps, err
}

func parseZipLockfile(f *zip.File) ([]reposource.PackageDependency, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	contents, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	ds, err := Parse(f.Name, contents)
	if err != nil {
		log15.Warn("failed to parse some lockfile dependencies", "error", err, "file", f.Name)
	}

	return ds, nil
}
