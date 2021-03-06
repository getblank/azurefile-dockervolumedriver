package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

var (
	recognizedOptions = []string{"share", "filemode", "dirmode", "uid", "gid", "nolock", "remotepath", "quota"}
)

type volumeMetadata struct {
	CreatedAt time.Time     `json:"created_at"`
	Account   string        `json:"account"`
	Options   VolumeOptions `json:"options"`
}

// VolumeOptions stores the opts passed to the driver by the docker engine.
type VolumeOptions struct {
	Share      string `json:"share"`
	FileMode   string `json:"filemode"`
	DirMode    string `json:"dirmode"`
	UID        string `json:"uid"`
	GID        string `json:"gid"`
	NoLock     bool   `json:"nolock"`
	RemotePath string `json:"remotepath"`
	Quota      string `json:"quota"`
}

type metadataDriver struct {
	metaDir string
}

func newMetadataDriver(metaDir string) (*metadataDriver, error) {
	if err := os.MkdirAll(metaDir, 0700); err != nil {
		return nil, fmt.Errorf("error creating %s: %v", metaDir, err)
	}
	return &metadataDriver{metaDir}, nil
}

func (m *metadataDriver) Validate(meta map[string]string) (volumeMetadata, error) {
	var v volumeMetadata
	var opts VolumeOptions

	// Validate keys
	for k := range meta {
		found := false
		for _, opts := range recognizedOptions {
			if k == opts {
				found = true
				break
			}
		}
		if !found {
			return v, fmt.Errorf("not a recognized volume driver option: %q", k)
		}
	}
	opts.Share = meta["share"]
	opts.DirMode = meta["dirmode"]
	opts.FileMode = meta["filemode"]
	opts.GID = meta["gid"]
	opts.UID = meta["uid"]
	opts.RemotePath = meta["remotepath"]
	opts.Quota = meta["quota"]

	if meta["nolock"] == "true" {
		opts.NoLock = true
	}

	return volumeMetadata{
		Options: opts,
	}, nil
}

func (m *metadataDriver) Delete(name string) error {
	if err := os.RemoveAll(m.path(name)); err != nil {
		return fmt.Errorf("cannot delete volume metadata: %v", err)
	}
	return nil
}

func (m *metadataDriver) Set(name string, meta volumeMetadata) error {
	b, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("cannot serialize metadata: %v", err)
	}
	if err := ioutil.WriteFile(m.path(name), b, 0600); err != nil {
		return fmt.Errorf("cannot write metadata: %v", err)
	}
	return nil
}

func (m *metadataDriver) Get(name string) (volumeMetadata, error) {
	var v volumeMetadata
	b, err := ioutil.ReadFile(m.path(name))
	if err != nil {
		return v, fmt.Errorf("cannot read metadata: %v", err)
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return v, fmt.Errorf("cannot deserialize metadata: %v", err)
	}
	return v, nil
}

func (m *metadataDriver) List() ([]string, error) {
	var volumes []string

	// return all the file names under metadata directory
	if err := filepath.Walk(m.metaDir, func(path string, info os.FileInfo, inErr error) error {
		if inErr != nil {
			return inErr
		}
		if path == m.metaDir {
			// directory itself, skip
			return nil
		}

		if info.IsDir() { // a directory
			return filepath.SkipDir
		}

		// base file name indicates the volume name
		volumes = append(volumes, filepath.Base(path))
		return nil
	}); err != nil {
		return volumes, fmt.Errorf("cannot list directory: %v", err)
	}
	return volumes, nil
}

func (m *metadataDriver) path(name string) string {
	return filepath.Join(m.metaDir, name)
}
