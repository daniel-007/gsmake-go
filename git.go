package gsmake

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsos"
	"github.com/gsdocker/gsos/uuid"
)

type gitSCM struct {
	gslogger.Log        // mixin log APIs
	cmd          string // command name
	name         string // command display name
}

func newGitSCM() (*gitSCM, error) {
	_, err := SearchCmd("git")

	if err != nil {
		return nil, err
	}

	return &gitSCM{
		Log:  gslogger.Get("gsmake"),
		cmd:  "git",
		name: "GIT",
	}, nil
}

func (git *gitSCM) String() string {
	return git.name
}

func (git *gitSCM) Cmd() string {
	return git.cmd
}

// Get implement SCM interface func
func (git *gitSCM) Get(url string, repopath string, version string, targetpath string) error {

	// if the local repo not exist, then clone it from host site
	if !gsos.IsDir(repopath) {

		// first clone package into cache dir
		cachedir := filepath.Join(os.TempDir(), uuid.NewV1().String())

		if err := os.MkdirAll(cachedir, 0755); err != nil {

			return err
		}

		if err := os.MkdirAll(filepath.Dir(repopath), 0755); err != nil {

			return err
		}

		command := exec.Command(git.cmd, "clone", url, cachedir)

		command.Stderr = os.Stderr
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout

		if err := command.Run(); err != nil {
			return err
		}

		if err := gsos.CopyDir(cachedir, repopath); err != nil {
			return err
		}
	}

	currentDir := gsos.CurrentDir()

	if err := os.Chdir(repopath); err != nil {
		return err
	}

	if version == "current" {
		version = "master"
	}

	command := exec.Command(git.name, "checkout", version)

	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout

	err := command.Run()

	os.Chdir(currentDir)

	if err != nil {
		return err
	}

	if gsos.IsExist(targetpath) {
		err := os.RemoveAll(targetpath)
		if err != nil {
			return err
		}
	}

	return gsos.CopyDir(repopath, targetpath)
}