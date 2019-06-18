// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	"github.com/AzureCR/acr-cli/cmd/api"
	"github.com/spf13/cobra"
)

const (
	purgeLongMessage = `` // TODO
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
		Short: "", // TODO
		Long:  purgeLongMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			var wg sync.WaitGroup
			auth := api.BasicAuth(username, password)
			var days int
			var hours int
			var minutes int
			_, e := fmt.Sscanf(ago, "%d.%d:%d", &days, &hours, &minutes)
			if e != nil || days < 0 || hours < 0 || hours > 23 || minutes < 0 || minutes > 59 {
				return errors.New("Please use the format dd:hh:mm and make sure that the specified Timespan is valid")
			}

			timeToCompare := time.Now().UTC()
			timeToCompare = timeToCompare.Add(time.Duration(-1*days) * 24 * time.Hour)
			timeToCompare = timeToCompare.Add(time.Duration(-1*hours) * time.Hour)
			timeToCompare = timeToCompare.Add(time.Duration(-1*minutes) * time.Minute)

			if !dangling {
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

	cmd.Flags().StringVar(&ago, "ago", "1.00:00", "The images that were created before this timeStamp will be deleted")
	cmd.Flags().BoolVar(&dangling, "dangling", false, "Just remove dangling manifests")
	cmd.Flags().StringVarP(&filter, "filter", "f", "", "Given as a regular expression, if a tag matches the pattern and is older than ago it gets deleted.")
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
		e := api.AcrDeleteTag(loginURL, auth, repoName, tag)
		if e != nil {
			return
		}
		fmt.Printf("%s/%s:%s\n", loginURL, repoName, tag)
	}
}

// DeleteManifest is the function in charge of deleting a manifest asynchronously
func DeleteManifest(wg *sync.WaitGroup, loginURL string, auth string, repoName string, digest string) {
	defer wg.Done()
	e := api.DeleteManifest(loginURL, auth, repoName, digest)
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Printf("%s/%s@%s\n", loginURL, repoName, digest)
}
