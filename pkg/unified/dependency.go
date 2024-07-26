package unified

import (
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "errors"

    "kcl-lang.io/kpm/pkg/downloader"
    "kcl-lang.io/kpm/pkg/git"
    "kcl-lang.io/kpm/pkg/oci"
    "kcl-lang.io/kpm/pkg/reporter"
    "kcl-lang.io/kpm/pkg/utils"
    "kcl-lang.io/kpm/pkg/3rdparty/par"
	remoteauth "oras.land/oras-go/v2/registry/remote/auth"
)

type DependencySystem struct {
    cache par.ErrCache[string, string]
    mu    sync.Mutex
}

func NewDependencySystem() *DependencySystem {
    return &DependencySystem{
        cache: par.ErrCache[string, string]{},
    }
}

func (ds *DependencySystem) Get(opts downloader.DownloadOptions) error {
    key := ds.generateCacheKey(opts.Source, opts.LocalPath)

    _, err := ds.cache.Do(key, func() (string, error) {
        if opts.Source.Git != nil {
            return ds.handleGitSource(opts)
        } else if opts.Source.Oci != nil || opts.Source.Registry != nil {
            return ds.handleOciSource(opts)
        }
        return "", fmt.Errorf("unsupported source type")
    })

    return err
}

func (ds *DependencySystem) generateCacheKey(source downloader.Source, localPath string) string {
    if source.Git != nil {
        return fmt.Sprintf("git:%s@%s:%s:%s", source.Git.Url, source.Git.Commit, source.Git.Tag, source.Git.Branch)
    } else if source.Oci != nil {
        return fmt.Sprintf("oci:%s/%s:%s", source.Oci.Reg, source.Oci.Repo, source.Oci.Tag)
    } else if source.Registry != nil {
        return fmt.Sprintf("registry:%s/%s:%s", source.Registry.Reg, source.Registry.Repo, source.Registry.Tag)
    }
    return fmt.Sprintf("unknown:%s", localPath)
}

func (ds *DependencySystem) handleGitSource(opts downloader.DownloadOptions) (string, error) {
    gitSource := opts.Source.Git
    _, err := git.CloneWithOpts(
        git.WithRepoURL(gitSource.Url),
        git.WithCommit(gitSource.Commit),
        git.WithTag(gitSource.Tag),
        git.WithBranch(gitSource.Branch),
        git.WithLocalPath(opts.LocalPath),
        git.WithWriter(opts.LogWriter),
    )
    if err != nil {
        return "", reporter.NewErrorEvent(
            reporter.FailedCloneFromGit,
            err,
            fmt.Sprintf("failed to clone from '%s' into '%s'.", gitSource.Url, opts.LocalPath),
        )
    }

    return opts.LocalPath, nil
}

func (ds *DependencySystem) handleOciSource(opts downloader.DownloadOptions) (string, error) {
    ociSource := opts.Source.Oci
    if ociSource == nil {
        return "", errors.New("oci source is nil")
    }

    localPath := opts.LocalPath

    repoPath := utils.JoinPath(ociSource.Reg, ociSource.Repo)

    var cred *remoteauth.Credential
    var err error
    if opts.CredsClient != nil {
        cred, err = opts.CredsClient.Credential(ociSource.Reg)
        if err != nil {
            return "", err
        }
    } else {
        cred = &remoteauth.Credential{}
    }

    ociCli, err := oci.NewOciClientWithOpts(
        oci.WithCredential(cred),
        oci.WithRepoPath(repoPath),
        oci.WithSettings(&opts.Settings),
    )

    if err != nil {
        return "", err
    }

    ociCli.PullOciOptions.Platform = opts.Platform

    if len(ociSource.Tag) == 0 {
        tagSelected, err := ociCli.TheLatestTag()
        if err != nil {
            return "", err
        }

        reporter.ReportMsgTo(
            fmt.Sprintf("the latest version '%s' will be downloaded", tagSelected),
            opts.LogWriter,
        )

        ociSource.Tag = tagSelected
    }

    reporter.ReportMsgTo(
        fmt.Sprintf(
            "downloading '%s:%s' from '%s/%s:%s'",
            ociSource.Repo, ociSource.Tag, ociSource.Reg, ociSource.Repo, ociSource.Tag,
        ),
        opts.LogWriter,
    )

    err = ociCli.Pull(localPath, ociSource.Tag)
    if err != nil {
        return "", err
    }

    matches, _ := filepath.Glob(filepath.Join(localPath, "*.tar"))
    if matches == nil || len(matches) != 1 {
        // then try to glob tgz file
        matches, _ = filepath.Glob(filepath.Join(localPath, "*.tgz"))
        if matches == nil || len(matches) != 1 {
            return "", fmt.Errorf("failed to find the downloaded kcl package tar file in '%s'", localPath)
        }
    }

    tarPath := matches[0]
    if utils.IsTar(tarPath) {
        err = utils.UnTarDir(tarPath, localPath)
    } else {
        err = utils.ExtractTarball(tarPath, localPath)
    }
    if err != nil {
        return "", fmt.Errorf("failed to untar the kcl package tar from '%s' into '%s'", tarPath, localPath)
    }

    // After untar the downloaded kcl package tar file, remove the tar file.
    if utils.DirExists(tarPath) {
        rmErr := os.Remove(tarPath)
        if rmErr != nil {
            return "", fmt.Errorf("failed to remove the downloaded kcl package tar file '%s'", tarPath)
        }
    }

    return localPath, nil
}