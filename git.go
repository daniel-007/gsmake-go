package gsmake

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsos"
	"github.com/gsdocker/gsos/uuid"
)

type gitSCM struct {
	gslogger.Log        // mixin log APIs
	homepath     string // gsmake home path
	cmd          string // command name
	name         string // command display name
}

func newGitSCM(homepath string) (*gitSCM, error) {
	_, err := SearchCmd("git")

	if err != nil {
		return nil, err
	}

	return &gitSCM{
		Log:      gslogger.Get("gsmake"),
		cmd:      "git",
		name:     "GIT",
		homepath: homepath,
	}, nil
}

func (git *gitSCM) String() string {
	return git.name
}

func (git *gitSCM) Cmd() string {
	return git.cmd
}

// Update implement SCM interface func
func (git *gitSCM) Update(url string, name string) error {
	repopath := RepoDir(git.homepath, name)

	if !gsos.IsDir(repopath) {
		return gserrors.Newf(ErrPackage, "package %s not cached", name)
	}

	currentDir := gsos.CurrentDir()

	if err := os.Chdir(repopath); err != nil {
		return gserrors.Newf(err, "git change current dir to work path error")
	}

	command := exec.Command(git.name, "pull")

	err := command.Run()

	os.Chdir(currentDir)

	if err != nil {
		return gserrors.Newf(err, "exec error :git pull")
	}

	return nil
}

func (git *gitSCM) Create(url string, name string, version string) (string, error) {
	repopath := RepoDir(git.homepath, name)

	// if the local repo not exist, then clone it from host site
	if !gsos.IsDir(repopath) {

		// first clone package into cache dir
		cachedir := filepath.Join(os.TempDir(), uuid.New())

		if err := os.MkdirAll(cachedir, 0755); err != nil {

			return repopath, err
		}

		if err := os.MkdirAll(filepath.Dir(repopath), 0755); err != nil {

			return repopath, err
		}

		command := exec.Command(git.cmd, "clone", url, cachedir)

		if err := command.Run(); err != nil {
			return repopath, err
		}

		if err := gsos.CopyDir(cachedir, repopath); err != nil {
			return repopath, err
		}
	}

	currentDir := gsos.CurrentDir()

	// fix windows git can't handle symlink repo directory bug
	realpath, err := os.Readlink(repopath)

	if err == nil {
		repopath = realpath
	}

	if err := os.Chdir(repopath); err != nil {
		return "", err
	}

	if version == "current" {
		version = "master"
	}

	command := exec.Command(git.name, "checkout", version)

	err = command.Run()

	os.Chdir(currentDir)

	if err != nil {
		return "", err
	}

	return repopath, nil
}

// Get implement SCM interface func
func (git *gitSCM) Get(url string, name string, version string, targetpath string) error {

	git.D("get package :%s", name)

	repopath, err := git.Create(url, name, version)

	if err != nil {
		return err
	}

	if gsos.IsExist(targetpath) {

		git.D("remove exist linked package :%s", targetpath)

		err := gsos.RemoveAll(targetpath)
		if err != nil {
			return gserrors.Newf(err, "git scm remove target dir error")
		}
	}

	return gsos.CopyDir(repopath, targetpath)
}
