package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/caarlos0/env"
	"github.com/tetsuo/fortune/frontend"
	"github.com/tetsuo/fortune/internal/wraperr"
	"golang.org/x/net/context/ctxhttp"
)

func frontendConfig(name, version, commit, date string) (cfg frontend.Config) {
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}
	if err := env.Parse(&cfg.DB); err != nil {
		panic(err)
	}
	cfg.Name = name
	cfg.VersionID = version
	cfg.VersionCommitHash = commit
	cfg.VersionCommitDate = date
	ctx := context.Background()
	if cfg.IsRunningOnGCE() {
		var err error
		cfg.ProjectID, err = getInstanceMetadata(ctx, "project/project-id")
		if err != nil {
			panic(err)
		}
		cfg.ZoneID, err = getInstanceMetadata(ctx, "instance/zone")
		if err != nil {
			panic(err)
		}
		cfg.ServiceAccount, err = getInstanceMetadata(ctx, "instance/service-accounts/default/email")
		if err != nil {
			panic(err)
		}
		cfg.InstanceID, err = getInstanceMetadata(ctx, "instance/id")
		if err != nil {
			panic(err)
		}
	} else {
		// dummy metadata for local development
		cfg.ProjectID = "dummy-gcp-project"
		cfg.ZoneID = "projects/823719028374/zones/us-central1-b"
		cfg.InstanceID = "9283746571029384756"
		cfg.ServiceAccount = "823719028374-compute@developer.gserviceaccount.com"
	}
	return
}

// getInstanceMetadata reads a metadata value from GCE.
// For the possible values of name, see:
// https://cloud.google.com/appengine/docs/standard/java/accessing-instance-metadata.
func getInstanceMetadata(ctx context.Context, name string) (_ string, err error) {
	defer wraperr.Wrap(&err, "getInstanceMetadata(ctx, %q)", name)
	req, err := http.NewRequest("GET", fmt.Sprintf("http://metadata.google.internal/computeMetadata/v1/%s", name), nil)
	if err != nil {
		return "", fmt.Errorf("http.NewRequest: %v", err)
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := ctxhttp.Do(ctx, nil, req)
	if err != nil {
		return "", fmt.Errorf("ctxhttp.Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("io.ReadAll: %v", err)
	}
	return string(bytes), nil
}
