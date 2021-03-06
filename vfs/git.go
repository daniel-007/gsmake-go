package vfs

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsdocker/gsos/fs"
	"github.com/gsdocker/gsos/uuid"
)

// ErrGitFS .
var (
	ErrGitFS = errors.New("git fs error")
)

// GitFS git fs for gsmake vfs
type GitFS struct {
	gslogger.Log // Mixin log APIs
}

// NewGitFS create new gitfs system
func NewGitFS() *GitFS {
	return &GitFS{
		Log: gslogger.Get("gitfs"),
	}
}

// Mount implement UserFS
func (gitFS *GitFS) String() string {
	return "git"
}

// Mount implement UserFS
func (gitFS *GitFS) Mount(rootfs RootFS, src, target *Entry) error {

	remote := src.Query().Get("remote")

	if remote == "" {
		return gserrors.Newf(ErrGitFS, "expect remoet url \n%s", src)
	}

	version := src.Query().Get("version")

	if version == "" {
		return gserrors.Newf(ErrGitFS, "expect remote repo version \n%s", src)
	}

	if version == "current" {
		version = "master"
	}

	cachepath := rootfs.CacheRoot(src)

	gitFS.D("mount remote url :%s", remote)

	gitFS.D("mount cache dir :%s", cachepath)

	// check if repo already exists

	if !fs.Exists(cachepath) {

		dirname := filepath.Base(uuid.New())

		rundir := os.TempDir()

		gitFS.I("cache package: %s:%s", filepath.Base(cachepath), version)

		startime := time.Now()

		if err := gitFS.clone(remote, rundir, dirname, true); err != nil {
			return gserrors.Newf(err, "clone cached repo error")
		}

		remote2 := filepath.Join(rundir, dirname)
		rundir = filepath.Dir(cachepath)
		dirname = filepath.Base(cachepath)

		if err := gitFS.clone(remote2, rundir, dirname, true); err != nil {
			return gserrors.Newf(err, "clone cached repo error")
		}

		if err := gitFS.setRemote(cachepath, "origin", remote); err != nil {
			return gserrors.Newf(err, "clone cached repo error")
		}

		gitFS.I("cache package -- success %s", time.Now().Sub(startime))

		if err := rootfs.Cached(src); err != nil {
			return err
		}
	}

	gitFS.D("mount target dir :%s", target.Mapping)

	rundir := filepath.Dir(target.Mapping)
	dirname := filepath.Base(target.Mapping)

	gitFS.I("clone cached package to userspace : %s", dirname)

	startime := time.Now()

	if err := gitFS.clone(cachepath, rundir, dirname, false); err != nil {
		return gserrors.Newf(err, "clone cached repo error")
	}

	gitFS.I("clone cached package to userspace -- success %s", time.Now().Sub(startime))

	// checkout version
	if err := gitFS.checkout(target.Mapping, version); err != nil {
		return gserrors.Newf(err, "checkout %s error", version)
	}

	return nil
}

func (gitFS *GitFS) clone(remote, rundir, dirname string, bare bool) error {

	if !fs.Exists(rundir) {
		if err := fs.MkdirAll(rundir, 0755); err != nil {
			return gserrors.Newf(err, "make clone target dir error")
		}
	}

	path := filepath.Join(rundir, dirname)

	if fs.Exists(path) {
		if err := fs.RemoveAll(path); err != nil {
			return gserrors.Newf(err, "remove exists repo error")
		}
	}

	var cmd *exec.Cmd

	cmd = exec.Command("git", "clone", remote, dirname)

	if !bare {
		cmd = exec.Command("git", "clone", remote, dirname)
	} else {
		cmd = exec.Command("git", "clone", "--mirror", remote, dirname)
	}

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	cmd.Dir = rundir

	return cmd.Run()
}

func (gitFS *GitFS) setRemote(rundir string, name string, url string) error {

	gitFS.D("change remote :\n\trepo:%s\n\tname:%s\n\turl:%s", rundir, name, url)

	cmd := exec.Command("git", "remote")

	var buff bytes.Buffer

	cmd.Stdout = &buff

	cmd.Stderr = os.Stderr

	cmd.Dir = rundir

	err := cmd.Run()

	if err != nil {
		return err
	}

	if !strings.Contains(buff.String(), name) {

		cmd = exec.Command("git", "remote", "add", name, url)

		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		cmd.Dir = rundir

		return cmd.Run()
	}

	cmd = exec.Command("git", "remote", "set-url", name, url)

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	cmd.Dir = rundir

	return cmd.Run()
}

func (gitFS *GitFS) fetch(rundir string) error {

	gitFS.D("git remote update :%s", rundir)

	cmd := exec.Command("git", "remote", "update")

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	cmd.Dir = rundir

	return cmd.Run()
}

func (gitFS *GitFS) pull(rundir string) error {

	cmd := exec.Command("git", "pull")

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	cmd.Dir = rundir

	return cmd.Run()
}

func (gitFS *GitFS) checkout(rundir string, version string) error {

	cmd := exec.Command("git", "checkout", version)

	var buff bytes.Buffer

	cmd.Stderr = &buff

	cmd.Dir = rundir

	if err := cmd.Run(); err != nil {
		return gserrors.Newf(err, buff.String())
	}

	return nil
}

// Dismount implement UserFS
func (gitFS *GitFS) Dismount(rootfs RootFS, src, target *Entry) error {

	gitFS.D("dismount dir :%s", target.Mapping)

	if fs.Exists(target.Mapping) {
		return fs.RemoveAll(target.Mapping)
	}

	return nil
}

// UpdateCache implement UserFS
func (gitFS *GitFS) UpdateCache(rootfs RootFS, cachepath string) error {

	gitFS.I("update cached package : %s", cachepath)

	startime := time.Now()

	if err := gitFS.fetch(filepath.Join(cachepath)); err != nil {
		return gserrors.Newf(err, "pull remote repo error")
	}

	gitFS.I("update cached package -- success %s", time.Now().Sub(startime))

	return nil
}

// Update implement UserFS
func (gitFS *GitFS) Update(rootfs RootFS, src, target *Entry, nocache bool) error {
	gserrors.Require(target.Scheme == FSGSMake, "target must be rootfs node")
	gserrors.Require(src.Scheme == "git", "src must be gitfs node")

	version := src.Query().Get("version")

	if version == "" {
		return gserrors.Newf(ErrGitFS, "expect remote repo version \n%s", src)
	}

	if version == "current" {
		version = "master"
	}

	cachepath := rootfs.CacheRoot(src)

	if nocache {

		if err := gitFS.UpdateCache(rootfs, cachepath); err != nil {
			return err
		}
	}

	rundir := filepath.Dir(target.Mapping)
	dirname := filepath.Base(target.Mapping)

	gitFS.D("clone rundir :%s ", rundir)
	gitFS.D("clone dirname :%s ", dirname)

	gitFS.I("clone cached package to userspace : %s", dirname)

	startime := time.Now()

	if err := gitFS.clone(cachepath, rundir, dirname, false); err != nil {
		return gserrors.Newf(err, "clone cached repo error")
	}
	gitFS.I("clone cached package to userspace -- success %s", time.Now().Sub(startime))

	// checkout version
	if err := gitFS.checkout(target.Mapping, version); err != nil {
		return gserrors.Newf(err, "checkout %s error", version)
	}

	return nil
}
