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

var loginURL string
var username string
var password string
var ago string
var dangling bool
var filter string
var repoName string
var maxEntries int

func newPurgeCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete images from a registry.",
		Long:  purgeLongMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			var wg sync.WaitGroup
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
				resultTags, e := api.ListTags(loginURL, auth, repoName, maxEntries)
				if e != nil {
					return e
				}
				tags := *resultTags.Tags
				for _, tag := range tags {
					//A regex filter was specified
					if len(filter) > 0 {
						matches, e := regexp.MatchString(filter, tag)
						if e != nil {
							return e
						}
						if !matches {
							continue
						}
					}

					wg.Add(1)
					go Untag(&wg, loginURL, auth, repoName, tag, timeToCompare)
				}
			}
			wg.Wait()
			resultManifests, e := api.AcrListManifests(loginURL, auth, repoName, "", "", maxEntries)
			if e != nil {
				return e
			}
			if resultManifests.Manifests != nil {
				manifests := *resultManifests.Manifests
				for _, manifest := range manifests {
					if manifest.Tags == nil {
						wg.Add(1)
						go DeleteManifest(&wg, loginURL, auth, repoName, *manifest.Digest)
					}
				}
			}
			wg.Wait()
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&loginURL, "registry", "r", "", "Registry login url")
	cmd.MarkPersistentFlagRequired("registry")
	cmd.PersistentFlags().StringVarP(&username, "username", "u", "", "Registry username")
	cmd.MarkPersistentFlagRequired("username")
	cmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Registry password")
	cmd.MarkPersistentFlagRequired("password")

	cmd.Flags().StringVar(&ago, "ago", "1d", "The images that were created before this timeStamp will be deleted")
	cmd.Flags().BoolVar(&dangling, "dangling", false, "Just remove dangling manifests")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "Given as a regular expression, if a tag matches the pattern and is older than the time specified in ago it gets deleted.")
	cmd.Flags().IntVar(&maxEntries, "max-entries", 100, "Maximum images to verify")
	cmd.Flags().StringVar(&repoName, "repository", "", "The repository which will be purged.")
	cmd.MarkFlagRequired("repository")

	return cmd
}

// Untag is the function responsible for untagging an image
func Untag(wg *sync.WaitGroup, loginURL string, auth string, repoName string, tag string, timeToCompare time.Time) {
	defer wg.Done()
	resultTagAttributes, e := api.AcrGetTagAttributes(loginURL, auth, repoName, tag)
	if e != nil {
		return
	}
	date := *(*resultTagAttributes.Tag).LastUpdateTime
	layout := time.RFC3339Nano
	t, e := time.Parse(layout, date)
	if e != nil {
		return
	}
	if t.Before(timeToCompare) {
		if e := api.AcrDeleteTag(loginURL, auth, repoName, tag); e != nil {
			return
		}
		fmt.Printf("%s/%s:%s\n", loginURL, repoName, tag)
	}
}

// DeleteManifest is the function in charge of deleting a manifest asynchronously
func DeleteManifest(wg *sync.WaitGroup, loginURL string, auth string, repoName string, digest string) {
	defer wg.Done()
	if e := api.DeleteManifest(loginURL, auth, repoName, digest); e != nil {
		return
	}
	fmt.Printf("%s/%s@%s\n", loginURL, repoName, digest)
}
