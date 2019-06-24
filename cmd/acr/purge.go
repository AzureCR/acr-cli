// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	acrapi "github.com/Azure/libacr/golang"
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
	archive      string
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
	cmd.Flags().StringVar(&archive, "archive-repository", "", "Instead of deleting manifests they will be moved to the repo specified here")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "Given as a regular expression, if a tag matches the pattern and is older than the time specified in ago it gets deleted")
	cmd.Flags().StringVar(&repoName, "repository", "", "The repository which will be purged")
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
				if len(archive) > 0 {
					var manifestMetadata *string
					manifestMetadata, e = api.AcrGetManifestMetadata(ctx, loginURL, auth, repoName, *tag.Digest, "acrarchiveinfo")
					if e != nil {
						//Metadata might be empty try initializing it
						tagMetadata := api.AcrTags{Name: tagName, ArchiveTime: timeToCompare.String()}
						tagsMetadataArray := make([]api.AcrTags, 0)
						metadataObject := &api.AcrManifestMetadata{Digest: *tag.Digest, OriginalRepo: repoName, Tags: append(tagsMetadataArray, tagMetadata)}
						var metadataBytes []byte
						metadataBytes, e = json.Marshal(metadataObject)
						if e != nil {
							return e
						}
						e = api.AcrUpdateManifestMetadata(ctx, loginURL, auth, repoName, *tag.Digest, "acrarchiveinfo", string(metadataBytes))
						if e != nil {
							return e
						}

					} else {
						//Existent metadata update it
						var metadataObject api.AcrManifestMetadata
						e = json.Unmarshal([]byte(*manifestMetadata), &metadataObject)
						if e != nil {
							return e
						}
						tagMetadata := api.AcrTags{Name: tagName, ArchiveTime: timeToCompare.String()}
						metadataObject.Tags = append(metadataObject.Tags, tagMetadata)
						var metadataBytes []byte
						metadataBytes, e = json.Marshal(metadataObject)
						if e != nil {
							return e
						}
						e = api.AcrUpdateManifestMetadata(ctx, loginURL, auth, repoName, *tag.Digest, "acrarchiveinfo", string(metadataBytes))
						if e != nil {
							return e
						}
					}
				}
				wg.Add(1)
				go Untag(ctx, wg, errorChannel, loginURL, auth, repoName, tagName)
			}
		}
		wg.Wait()
		for len(errorChannel) > 0 {
			e = <-errorChannel
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
func PurgeDanglingManifests(ctx context.Context, wg *sync.WaitGroup, loginURL string, auth string, repoName string) error {
	lastManifestDigest := ""
	resultManifests, e := api.AcrListManifests(ctx, loginURL, auth, repoName, "", lastManifestDigest)
	if e != nil {
		return e
	}
	for resultManifests.Manifests != nil && e == nil {
		manifests := *resultManifests.Manifests
		for _, manifest := range manifests {
			if manifest.Tags == nil {
				wg.Add(1)
				go HandleManifest(ctx, wg, manifest, loginURL, auth, repoName)
			}
		}
		lastManifestDigest = *manifests[len(manifests)-1].Digest
		resultManifests, e = api.AcrListManifests(ctx, loginURL, auth, repoName, "", lastManifestDigest)
	}
	return nil
}

// HandleManifest deletes a manifest, if there is an archive repo and the manifest has existent metadata the manifest is moved instead.
func HandleManifest(ctx context.Context, wg *sync.WaitGroup, manifest acrapi.ManifestAttributesBase, loginURL string, auth, repoName string) {
	defer wg.Done()
	var e error
	if len(archive) > 0 {
		var manifestMetadata *string
		manifestMetadata, e = api.AcrGetManifestMetadata(ctx, loginURL, auth, repoName, *manifest.Digest, "acrarchiveinfo")
		// if there is an error getting the metadata the manifest gets deleted with no cross repository mounting.
		if e == nil {
			var metadataObject api.AcrManifestMetadata
			e = json.Unmarshal([]byte(*manifestMetadata), &metadataObject)
			if e != nil {
				return
			}
			//Tags empty len 0
			var manifestV2 *api.ManifestV2
			manifestV2, e = api.GetManifest(ctx, loginURL, auth, repoName, *manifest.Digest)
			if e != nil {
				return
			}
			e = api.AcrCrossReferenceLayer(ctx, loginURL, auth, archive, *(*manifestV2.Config).Digest, repoName)
			if e != nil {
				return
			}
			for _, layer := range *manifestV2.Layers {
				e = api.AcrCrossReferenceLayer(ctx, loginURL, auth, archive, *layer.Digest, repoName)
				if e != nil {
					return
				}
			}
			newTagName := repoName + (*manifest.Digest)[len("sha256:"):len("sha256:")+8]
			e = api.PutManifest(ctx, loginURL, auth, archive, newTagName, *manifestV2)
			if e != nil {
				return
			}
			e = api.AcrUpdateTagMetadata(ctx, loginURL, auth, archive, newTagName, "acrarchiveinfo", *manifestMetadata)
			if e != nil {
				return
			}
		}
	}
	if e := api.DeleteManifest(ctx, loginURL, auth, repoName, *manifest.Digest); e != nil {
		return
	}
	fmt.Printf("%s/%s@%s\n", loginURL, repoName, *manifest.Digest)
}
