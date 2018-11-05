package main

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"syscall"

	g8ufs "github.com/threefoldtech/0-fs"
	"github.com/threefoldtech/0-fs/meta"
	"github.com/threefoldtech/0-fs/storage"
)

//a helper to close all under laying readers in a flist file stream since decompression doesn't
//auto close the under laying layer.
type underLayingCloser struct {
	readers []io.Reader
}

//close all layers.
func (u *underLayingCloser) Close() error {
	for i := len(u.readers) - 1; i >= 0; i-- {
		r := u.readers[i]
		if c, ok := r.(io.Closer); ok {
			c.Close()
		}
	}

	return nil
}

//read only from the last layer.
func (u *underLayingCloser) Read(p []byte) (int, error) {
	return u.readers[len(u.readers)-1].Read(p)
}

func getMetaDBTar(src string) (io.ReadCloser, error) {
	u, err := url.Parse(src)
	if err != nil {
		return nil, err
	}

	var reader io.ReadCloser
	base := path.Base(u.Path)

	if u.Scheme == "file" || u.Scheme == "" {
		// check file exists
		_, err := os.Stat(u.Path)
		if err != nil {
			return nil, err
		}
		reader, err = os.Open(u.Path)
		if err != nil {
			return nil, err
		}
	} else if u.Scheme == "http" || u.Scheme == "https" {
		response, err := http.Get(src)
		if err != nil {
			return nil, err
		}

		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download flist: %s", response.Status)
		}

		reader = response.Body
	} else {
		return nil, fmt.Errorf("invalid flist url (%s)", src)
	}

	var closer underLayingCloser
	closer.readers = append(closer.readers, reader)

	ext := path.Ext(base)
	switch ext {
	case ".tgz":
		fallthrough
	case ".flist":
		fallthrough
	case ".gz":
		if r, err := gzip.NewReader(reader); err != nil {
			closer.Close()
			return nil, err
		} else {
			closer.readers = append(closer.readers, r)
		}
		return &closer, nil
	case ".tbz2":
		fallthrough
	case ".bz2":
		closer.readers = append(closer.readers, bzip2.NewReader(reader))
		return &closer, err
	case ".tar":
		return &closer, nil
	}

	return nil, fmt.Errorf("unknown flist format %s", ext)
}

func getMetaDB(namespace, src string) (string, error) {
	reader, err := getMetaDBTar(src)
	if err != nil {
		return "", err
	}

	defer reader.Close()

	archive := tar.NewReader(reader)
	db := fmt.Sprintf("%s.db", namespace)
	if err := os.MkdirAll(db, 0755); err != nil {
		return "", err
	}

	for {
		header, err := archive.Next()
		if err != nil && err != io.EOF {
			return "", err
		} else if err == io.EOF {
			break
		}

		if header.FileInfo().IsDir() {
			continue
		}

		base := path.Join(db, path.Dir(header.Name))
		if err := os.MkdirAll(base, 0755); err != nil {
			return "", err
		}

		file, err := os.Create(path.Join(db, header.Name))
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(file, archive); err != nil {
			file.Close()
			return "", err
		}

		file.Close()
	}

	return db, nil
}

//Chroot builds an flist chroot mount
type Chroot struct {
	ID      string
	Flist   string
	Storage string

	fs *g8ufs.G8ufs
}

func (c *Chroot) getBaseDir(p ...string) string {
	user, err := user.Current()
	if err != nil {
		panic("failed to get current user")
	}

	return path.Join(user.HomeDir, path.Join(p...))
}

//MountRoot returns the mountpoint path
func (c *Chroot) MountRoot() string {
	return c.getBaseDir(".zbundle", c.ID)
}

//WorkDirRoot returns the working directory root of an flist
func (c *Chroot) WorkDirRoot() string {
	return c.getBaseDir(".zbundle.wd", c.ID)
}

func (c *Chroot) prepare() error {
	root := c.MountRoot()
	for _, dir := range []string{"proc", "dev", "sys"} {
		target := path.Join(root, dir)
		os.MkdirAll(target, 0755)
		if err := syscall.Mount(path.Join("/", dir), target, "", syscall.MS_BIND, ""); err != nil {
			return err
		}
	}

	//NOTE: the following secion is optional. We should not crash
	//if we failed to copy the resolv.conf from host to bundle
	os.MkdirAll(path.Join(root, "etc"), 0755)
	resolve, err := ioutil.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}

	ioutil.WriteFile(path.Join(root, "etc", "resolv.conf"), resolve, 0644)
	return nil
}

func (c *Chroot) unPrepare() {
	root := c.MountRoot()
	for _, dir := range []string{"proc", "dev", "sys"} {
		target := path.Join(root, dir)
		syscall.Unmount(target, syscall.MNT_FORCE|syscall.MNT_DETACH)
	}
}

//isMount checks if path is a mountpoint
func isMount(path string) bool {
	//TODO: use a better approach to check if a mount point
	//TODO: inotify ?
	cmd := exec.Command("mountpoint", "-q", path)
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

//Start starts the chroot
func (c *Chroot) Start() error {
	root := c.MountRoot()

	if isMount(root) {
		return fmt.Errorf("a chroot is running with the same id")
	}

	os.MkdirAll(root, 0755)
	// should we do this under temp?
	namespace := c.WorkDirRoot()

	metaPath, err := getMetaDB(namespace, c.Flist)
	if err != nil {
		return err
	}

	metaStore, err := meta.NewStore(metaPath)
	if err != nil {
		return err
	}

	stor, err := storage.NewSimpleStorage(c.Storage)
	if err != nil {
		return err
	}

	opt := g8ufs.Options{
		Backend:   namespace,
		Target:    root,
		MetaStore: metaStore,
		Storage:   stor,
		Cache:     path.Join(c.getBaseDir(".zbundle.cache")),
	}

	fs, err := g8ufs.Mount(&opt)
	if err != nil {
		return err
	}

	c.fs = fs
	return c.prepare()
}

//Stop stops the chroot
func (c *Chroot) Stop() error {
	if c.fs == nil {
		return fmt.Errorf("chroot is not started")
	}

	namespace := c.WorkDirRoot()

	//we only remove the meta director (the one contains the database)
	//so zbundles can be resumed
	defer os.RemoveAll(fmt.Sprintf("%s.db", namespace))
	c.unPrepare()
	return c.fs.Unmount()
}

//Wait for chroot to terminate
func (c *Chroot) Wait() error {
	return c.fs.Wait()
}
