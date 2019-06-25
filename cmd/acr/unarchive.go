// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/AzureCR/acr-cli/cmd/api"
	"github.com/spf13/cobra"
)

const (
	unarchiveLongMessage = `` // TODO
)

var reference string
var newTagName string

func newUnarchiveCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unarchive",
		Short: "acr unarchive: restore an image deleted by acr purge.", // TODO
		Long:  unarchiveLongMessage,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			loginURL := api.LoginURL(registryName)
			auth := api.BasicAuth(username, password)
			if !strings.HasPrefix(reference, "sha256") {
				return errors.New("reference has to be a digest")
			}
			tagName := repoName + reference[len("sha256:"):len("sha256:")+8]
			tagMetadata, e := api.AcrGetTagMetadata(ctx, loginURL, auth, archive, tagName, "acrarchiveinfo")
			if e != nil {
				return e
			}
			var metadataObject api.AcrManifestMetadata
			e = json.Unmarshal([]byte(*tagMetadata), &metadataObject)
			if e != nil {
				return e
			}
			var manifestV2 *api.ManifestV2
			manifestV2, e = api.GetManifest(ctx, loginURL, auth, archive, tagName)
			if e != nil {
				return e
			}
			e = api.AcrCrossReferenceLayer(ctx, loginURL, auth, repoName, *(*manifestV2.Config).Digest, archive)
			if e != nil {
				return e
			}
			for _, layer := range *manifestV2.Layers {
				e = api.AcrCrossReferenceLayer(ctx, loginURL, auth, repoName, *layer.Digest, archive)
				if e != nil {
					return e
				}
			}

			if len(newTagName) > 0 {
				e = api.PutManifest(ctx, loginURL, auth, repoName, newTagName, *manifestV2)
				if e != nil {
					return e
				}
				fmt.Println(newTagName)
			} else {
				for _, tag := range metadataObject.Tags {
					e = api.PutManifest(ctx, loginURL, auth, repoName, tag.Name, *manifestV2)
					if e != nil {
						return e
					}
					fmt.Println(tag.Name)
				}
			}
			tagInfo, e := api.AcrGetTagAttributes(ctx, loginURL, auth, archive, tagName)
			if e != nil {
				return e
			}
			e = api.DeleteManifest(ctx, loginURL, auth, archive, *(*tagInfo.Tag).Digest)
			if e != nil {
				return e
			}
			return nil
		},
	}

	cmd.MarkPersistentFlagRequired("archive-repository")
	cmd.Flags().StringVar(&reference, "reference", "", "Either a digest")
	cmd.MarkFlagRequired("reference")
	cmd.Flags().StringVar(&newTagName, "tag-name", "", "Either a digest")

	return cmd
}
