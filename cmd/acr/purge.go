// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/AzureCR/acr-cli/cmd/api"
	"github.com/spf13/cobra"
)

const (
	purgeLongMessage = `acr purge: untag old images and delete dangling manifests.`
	exampleMessage   = `
Delete all tags that are older than 1 day
  acr purge -r MyRegistry --repository MyRepository --ago 1d

Delete all tags that are older than 1 day and begin with hello
  acr purge -r MyRegistry --repository MyRepository --ago 1d --filter "^hello.*"

Delete all dangling manifests
  acr purge -r MyRegistry --repository MyRepository --dangling`
)

var (
	registryName string
	username     string
	password     string
	ago          string
	dangling     bool
	filter       string
	repoName     string
)

func newPurgeCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "purge",
		Short:   "Delete images from a registry.",
		Long:    purgeLongMessage,
		Example: exampleMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var e error
			var wg sync.WaitGroup
			loginURL := api.LoginURL(registryName)
			auth := api.BasicAuth(username, password)
			if !dangling {
				e = PurgeTags(ctx, &wg, loginURL, auth, repoName)
				if e != nil {
					return e
				}
			}
			e = PurgeDanglingManifests(ctx, &wg, loginURL, auth, repoName)
			if e != nil {
				return e
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&registryName, "registry", "r", "", "Registry name")
	cmd.MarkPersistentFlagRequired("registry")
	cmd.PersistentFlags().StringVarP(&username, "username", "u", "", "Registry username")
	cmd.MarkPersistentFlagRequired("username")
	cmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Registry password")
	cmd.MarkPersistentFlagRequired("password")

	cmd.Flags().StringVar(&ago, "ago", "1d", "The images that were created before this timeStamp will be deleted")
	cmd.Flags().BoolVar(&dangling, "dangling", false, "Just remove dangling manifests")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "Given as a regular expression, if a tag matches the pattern and is older than the time specified in ago it gets deleted.")
	cmd.Flags().StringVar(&repoName, "repository", "", "The repository which will be purged.")
	cmd.MarkFlagRequired("repository")

	return cmd
}

// PurgeTags deletes all tags that are older than the ago value and that match the filter string (if present)
func PurgeTags(ctx context.Context, wg *sync.WaitGroup, loginURL string, auth string, repoName string) error {
	var days int
	var durationString string
	if strings.Contains(ago, "d") {
		if _, e := fmt.Sscanf(ago, "%dd%s", &days, &durationString); e != nil {
			fmt.Sscanf(ago, "%dd", &days)
			durationString = ""
		}
	} else {
		days = 0
		if _, e := fmt.Sscanf(ago, "%s", &durationString); e != nil {
			return e
		}
	}
	timeToCompare := time.Now().UTC()
	timeToCompare = timeToCompare.Add(time.Duration(-1*days) * 24 * time.Hour)
	if len(durationString) > 0 {
		agoDuration, e := time.ParseDuration(durationString)
		if e != nil {
			return e
		}
		timeToCompare = timeToCompare.Add(-1 * agoDuration)
	}
	var matches bool
	var t time.Time
	var errorChannel = make(chan error, 100)
	defer close(errorChannel)
	lastTag := ""
	resultTags, e := api.AcrListTags(ctx, loginURL, auth, repoName, "", lastTag)
	for resultTags != nil && resultTags.Tags != nil && e == nil {
		tags := *resultTags.Tags
		for _, tag := range tags {
			tagName := *tag.Name
			//A regex filter was specified
			if len(filter) > 0 {
				matches, e = regexp.MatchString(filter, tagName)
				if e != nil {
					return e
				}
				if !matches {
					continue
				}
			}
			lastUpdateTime := *tag.LastUpdateTime
			layout := time.RFC3339Nano
			t, e = time.Parse(layout, lastUpdateTime)
			if e != nil {
				return e
			}
			if t.Before(timeToCompare) {
				wg.Add(1)
				go Untag(ctx, wg, errorChannel, loginURL, auth, repoName, tagName)
			}
		}
		wg.Wait()
		for len(errorChannel) > 0 {
			e := <-errorChannel
			if e != nil {
				return e
			}
		}
		lastTag = *tags[len(tags)-1].Name
		resultTags, e = api.AcrListTags(ctx, loginURL, auth, repoName, "", lastTag)
		if e != nil {
			return e
		}
	}
	return nil
}

// Untag is the function responsible for untagging an image
func Untag(ctx context.Context,
	wg *sync.WaitGroup,
	errorChannel chan error,
	loginURL string,
	auth string,
	repoName string,
	tag string) {
	defer wg.Done()
	e := api.AcrDeleteTag(ctx, loginURL, auth, repoName, tag)
	if e != nil {
		errorChannel <- e
		return
	}
	fmt.Printf("%s/%s:%s\n", loginURL, repoName, tag)
}

// PurgeDanglingManifests runs if the dangling flag is specified and deletes all manifests that do not have any tags associated with them.
func PurgeDanglingManifests(ctx context.Context,
	wg *sync.WaitGroup,
	loginURL string,
	auth string,
	repoName string) error {
	var errorChannel = make(chan error, 100)
	defer close(errorChannel)
	lastManifestDigest := ""
	resultManifests, e := api.AcrListManifests(ctx, loginURL, auth, repoName, "", lastManifestDigest)
	if e != nil {
		return e
	}
	for resultManifests != nil && resultManifests.Manifests != nil && e == nil {
		manifests := *resultManifests.Manifests
		for _, manifest := range manifests {
			if manifest.Tags == nil {
				wg.Add(1)
				go HandleManifest(ctx, wg, errorChannel, loginURL, auth, repoName, *manifest.Digest)
			}
		}
		wg.Wait()
		for len(errorChannel) > 0 {
			e = <-errorChannel
			if e != nil {
				return e
			}
		}
		lastManifestDigest = *manifests[len(manifests)-1].Digest
		resultManifests, e = api.AcrListManifests(ctx, loginURL, auth, repoName, "", lastManifestDigest)
		if e != nil {
			return e
		}
	}
	return nil
}

// HandleManifest deletes a manifest, if there is an archive repo and the manifest has existent metadata the manifest is moved instead.
func HandleManifest(ctx context.Context,
	wg *sync.WaitGroup,
	errorChannel chan error,
	loginURL string,
	auth string,
	repoName string,
	digest string) {
	defer wg.Done()
	e := api.DeleteManifest(ctx, loginURL, auth, repoName, digest)
	if e != nil {
		errorChannel <- e
		return
	}
	fmt.Printf("%s/%s@%s\n", loginURL, repoName, digest)
}
