// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
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
	purgeLongMessage = `Purge the registry, given the registry name and a repository name this 
command untags all the tags that match with the filter and that are older 
than a duration, after that, all manifests that do not have any tags 
associated with them also get deleted.`
)

var registryName string
var username string
var password string
var ago string
var dangling bool
var filter string
var repoName string

func newPurgeCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete images from a registry.",
		Long:  purgeLongMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			var wg sync.WaitGroup
			loginURL := api.LoginURL(registryName)
			auth := api.BasicAuth(username, password)
			if !dangling {
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
				lastTag := ""
				resultTags, e := api.AcrListTags(loginURL, auth, repoName, "", lastTag)
				for resultTags.Tags != nil && e == nil {
					tags := *resultTags.Tags
					for _, tag := range tags {
						tagName := *tag.Name
						//A regex filter was specified
						if len(filter) > 0 {
							if matches, e := regexp.MatchString(filter, tagName); e == nil {
								if !matches {
									continue
								}
							} else {
								return e
							}
						}
						createdTime := *tag.LastUpdateTime
						layout := time.RFC3339Nano
						if t, e := time.Parse(layout, createdTime); e == nil {
							if t.Before(timeToCompare) {
								wg.Add(1)
								go Untag(&wg, loginURL, auth, repoName, tagName)
							}
						} else {
							return e
						}
					}
					lastTag = *tags[len(tags)-1].Name
					resultTags, e = api.AcrListTags(loginURL, auth, repoName, "", lastTag)
				}
			}
			wg.Wait()
			lastManifestDigest := ""
			for resultManifests, e := api.AcrListManifests(loginURL, auth, repoName, "", lastManifestDigest); resultManifests.Manifests != nil && e == nil; {
				manifests := *resultManifests.Manifests
				for _, manifest := range manifests {
					if manifest.Tags == nil {
						wg.Add(1)
						go DeleteManifest(&wg, loginURL, auth, repoName, *manifest.Digest)
					}
				}
				lastManifestDigest = *manifests[len(manifests)-1].Digest
				resultManifests, e = api.AcrListManifests(loginURL, auth, repoName, "", lastManifestDigest)
			}
			wg.Wait()
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

// Untag is the function responsible for untagging an image
func Untag(wg *sync.WaitGroup, loginURL string, auth string, repoName string, tag string) {
	defer wg.Done()
	if e := api.AcrDeleteTag(loginURL, auth, repoName, tag); e != nil {
		return
	}
	fmt.Printf("%s/%s:%s\n", loginURL, repoName, tag)

}

// DeleteManifest is the function in charge of deleting a manifest asynchronously
func DeleteManifest(wg *sync.WaitGroup, loginURL string, auth string, repoName string, digest string) {
	defer wg.Done()
	if e := api.DeleteManifest(loginURL, auth, repoName, digest); e != nil {
		return
	}
	fmt.Printf("%s/%s@%s\n", loginURL, repoName, digest)
}
