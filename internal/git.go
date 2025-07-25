package internal

import (
	"io"
	"os"
	"strings"

	"github.com/blang/semver/v4"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/sirupsen/logrus"
)

const (
	versionFilePath = "VERSION.txt"
)

func (g *GitClient) GetVersion(repoURL, commitHash string) (*semver.Version, error) {
	err := g.cloneRepository(repoURL)
	if err != nil {
		return nil, err
	}
	// Resolve the commit hash
	hash := plumbing.NewHash(commitHash)
	repo := normalizeRepoURL(repoURL)
	commit, err := g.refs[repo].repo.CommitObject(hash)
	if err != nil {
		return nil, err
	}

	// Get the tree from the commit
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	// Find the file
	file, err := tree.File(versionFilePath)
	if err != nil {
		return nil, err
	}
	// Read the contents
	reader, err := file.Blob.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	// Get file contents at given ref
	sver, err := semver.ParseTolerant(string(content))
	if err != nil {
		return nil, err
	}
	return &sver, nil

}

func normalizeRepoURL(repoURL string) string {
	if strings.HasSuffix(repoURL, ".git") {
		return strings.TrimSuffix(repoURL, ".git")
	}
	return repoURL

}
func (g *GitClient) cloneRepository(repoURL string) error {
	repo := normalizeRepoURL(repoURL)
	if _, ok := g.refs[repo]; ok {
		return nil
	}
	// Clone the remote repository into a temporary directory
	tmpDir, err := os.MkdirTemp("", "go-git-repo-*")
	if err != nil {
		return err
	}
	logrus.Debugf("Cloning repository %s", repoURL)
	r, err := git.PlainClone(tmpDir, false, &git.CloneOptions{URL: repoURL, ReferenceName: plumbing.HEAD})
	if err != nil {
		return err
	}
	g.refs[repo] = repositoryReference{file: tmpDir, repo: r}
	return nil
}

type GitCommitVersioner interface {
	GetVersion(commitHash, filePath string) (*semver.Version, error)
	Cleanup()
}

type GitClient struct {
	refs map[string]repositoryReference
}

type repositoryReference struct {
	repo *git.Repository
	file string
}

func NewGitClient() GitCommitVersioner {
	return &GitClient{
		refs: make(map[string]repositoryReference),
	}
}
func (g *GitClient) Cleanup() {
	for k, v := range g.refs {
		logrus.Debugf("Cleaning up git clone directory for repo %s on %s", k, v.file)
		if err := os.RemoveAll(k); err != nil {
			logrus.Error(os.RemoveAll(v.file))
		}
	}
}
