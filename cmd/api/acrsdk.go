// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	acrapi "github.com/Azure/libacr/golang"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

const (
	prefixHTTPS = "https://"
	registryURL = ".azurecr.io"
)

// BasicAuth returns the username and the passwrod encoded in base 64
func BasicAuth(username string, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// LoginURL returns the FQDN for a registry
func LoginURL(registryName string) string {
	// TODO: if the registry is in another cloud (i.e. dogfood) a full FQDN for the registry should be specified.
	if strings.Contains(registryName, ".") {
		return registryName
	}
	return registryName + registryURL
}

// GetHostname return the hostname of a registry
func GetHostname(loginURL string) string {
	hostname := loginURL
	if !strings.HasPrefix(loginURL, prefixHTTPS) {
		hostname = prefixHTTPS + loginURL
	}
	return hostname
}

// AcrListTags list the tags of a repository with their attributes
func AcrListTags(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	orderBy string,
	last string) (*acrapi.TagAttributeList, error) {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		"",
		"",
		"",
		"",
		auth,
		orderBy,
		"100",
		last,
		"")
	if tags, e := client.AcrListTags(ctx); e == nil {
		var listTagResult acrapi.TagAttributeList
		switch tags.StatusCode {
		case http.StatusOK:
			if e = mapstructure.Decode(tags.Value, &listTagResult); e == nil {
				return &listTagResult, nil
			}
			return nil, e

		case http.StatusUnauthorized, http.StatusNotFound:
			var apiError acrapi.Error
			if e = mapstructure.Decode(tags.Value, &apiError); e == nil {
				return nil, fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return nil, errors.Wrap(e, "unable to decode error")

		default:
			return nil, fmt.Errorf("unexpected response code: %v", tags.StatusCode)
		}
	} else {
		return nil, e
	}
}

// AcrDeleteTag deletes the tag by reference.
func AcrDeleteTag(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string) error {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		"",
		"",
		auth,
		"",
		"",
		"",
		"")

	if tag, e := client.AcrDeleteTag(ctx); e == nil {
		switch tag.StatusCode {
		case http.StatusAccepted:
			return nil
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if e = mapstructure.Decode(tag, &apiError); e == nil {
				return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return errors.Wrap(e, "unable to decode error")

		default:
			return fmt.Errorf("unexpected response code: %v", tag.StatusCode)
		}
	} else {
		return e
	}
}

// AcrListManifests list all the manifest in a repository with their attributes.
func AcrListManifests(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	orderBy string,
	last string) (*acrapi.ManifestAttributeList, error) {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		"",
		"",
		"",
		"",
		auth,
		orderBy,
		"100",
		last,
		"")

	if manifests, e := client.AcrListManifests(ctx); e == nil {
		switch manifests.StatusCode {
		case http.StatusOK:
			var acrListManifestsAttributesResult acrapi.ManifestAttributeList
			if e = mapstructure.Decode(manifests.Value, &acrListManifestsAttributesResult); e == nil {
				return &acrListManifestsAttributesResult, nil
			}
			return nil, e

		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if e = mapstructure.Decode(manifests.Value, &apiError); e == nil {
				return nil, fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return nil, errors.Wrap(e, "unable to decode error")

		default:
			return nil, fmt.Errorf("unexpected response code: %v", manifests.StatusCode)
		}
	} else {
		return nil, e
	}
}

// DeleteManifest deletes a manifest using the digest as a reference.
func DeleteManifest(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string) error {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		"",
		"",
		auth,
		"",
		"",
		"",
		"")

	if deleteManifest, e := client.DeleteManifest(ctx); e == nil {
		switch deleteManifest.StatusCode {
		case http.StatusAccepted:
			return nil
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if e = mapstructure.Decode(deleteManifest, &apiError); e == nil {
				return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return errors.Wrap(e, "unable to decode error")

		default:
			return fmt.Errorf("unexpected response code: %v", deleteManifest.StatusCode)
		}
	} else {
		return e
	}
}

// AcrGetManifestMetadata get the metadata of a manifest
func AcrGetManifestMetadata(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string,
	metadataName string) (*string, error) {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		metadataName,
		"",
		auth,
		"",
		"",
		"",
		"")

	if tagMetadata, e := client.AcrGetManifestMetadata(ctx); e == nil {
		var acrGetTagMetadataResult string
		switch tagMetadata.StatusCode {
		case http.StatusOK:
			if e = mapstructure.Decode(tagMetadata.Value, &acrGetTagMetadataResult); e == nil {
				return &acrGetTagMetadataResult, nil
			}
			return nil, e
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if e = mapstructure.Decode(tagMetadata.Value, &apiError); e == nil {
				return nil, fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return nil, errors.Wrap(e, "unable to decode error")
		default:
			return nil, fmt.Errorf("unexpected response code: %v", tagMetadata.StatusCode)
		}
	} else {
		return nil, e
	}
}

// AcrUpdateManifestMetadata create or update a manifest metadata
func AcrUpdateManifestMetadata(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string,
	metadataName string,
	value string) error {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		metadataName,
		"",
		auth,
		"",
		"",
		"",
		"")

	if manifestMetadata, e := client.AcrUpdateManifestMetadata(ctx, value); e == nil {
		switch manifestMetadata.StatusCode {
		case http.StatusCreated:
			return nil
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var metadataError acrapi.Error
			if e = mapstructure.Decode(manifestMetadata, &metadataError); e == nil {
				return fmt.Errorf("%s %s", *(*metadataError.Errors)[0].Code, *(*metadataError.Errors)[0].Message)
			}
			return e
		default:
			return fmt.Errorf("unexpected response code: %v", manifestMetadata.StatusCode)
		}
	} else {
		return e
	}
}

// AcrGetTagMetadata get the metadata of a manifest
func AcrGetTagMetadata(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string,
	metadataName string) (*string, error) {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		metadataName,
		"",
		auth,
		"",
		"",
		"",
		"")

	if manifestMetadata, e := client.AcrGetTagMetadata(ctx); e == nil {
		var acrGetManifestMetadataResult string
		switch manifestMetadata.StatusCode {
		case http.StatusOK:
			if e = mapstructure.Decode(manifestMetadata.Value, &acrGetManifestMetadataResult); e == nil {
				return &acrGetManifestMetadataResult, nil
			}
			return nil, e
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if e = mapstructure.Decode(manifestMetadata.Value, &apiError); e == nil {
				return nil, fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return nil, errors.Wrap(e, "unable to decode error")
		default:
			return nil, fmt.Errorf("unexpected response code: %v", manifestMetadata.StatusCode)
		}
	} else {
		return nil, e
	}
}

// AcrUpdateTagMetadata updates or creates metadata for a tag
func AcrUpdateTagMetadata(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string,
	metadataName string,
	value string) error {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		metadataName,
		"",
		auth,
		"",
		"",
		"",
		"")

	if tagMetadata, e := client.AcrUpdateTagMetadata(ctx, value); e == nil {
		switch tagMetadata.StatusCode {
		case http.StatusCreated:
			return nil
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var metadataError acrapi.Error
			if e = mapstructure.Decode(tagMetadata, &metadataError); e == nil {
				return fmt.Errorf("%s %s", *(*metadataError.Errors)[0].Code, *(*metadataError.Errors)[0].Message)
			}
			return e
		default:
			return fmt.Errorf("unexpected response code: %v", tagMetadata.StatusCode)
		}
	} else {
		return e
	}
}

// GetManifest returns the V2 manifest schema
func GetManifest(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string) (*ManifestV2, error) {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		"",
		"application/vnd.docker.distribution.manifest.v2+json",
		auth,
		"",
		"",
		"",
		"")

	if manifest, e := client.GetManifest(ctx); e == nil {
		var getManifestResult ManifestV2
		switch manifest.StatusCode {
		case http.StatusOK:
			if e = mapstructure.Decode(manifest.Value, &getManifestResult); e == nil {
				return &getManifestResult, nil
			}
			return nil, e
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound:
			var metadataError acrapi.Error
			if e = mapstructure.Decode(manifest.Value, &metadataError); e == nil {
				return nil, fmt.Errorf("%s %s", *(*metadataError.Errors)[0].Code, *(*metadataError.Errors)[0].Message)
			}
			return nil, errors.Wrap(e, "unable to decode error")
		default:
			return nil, fmt.Errorf("unexpected response code: %v", manifest.StatusCode)
		}
	} else {
		return nil, e
	}
}

// AcrCrossReferenceLayer ...
func AcrCrossReferenceLayer(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string,
	repoFrom string) error {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		"",
		"",
		auth,
		"",
		"",
		"",
		"")

	var result acrapi.SetObject
	pathParameters := map[string]interface{}{
		"name": autorest.Encode("path", client.Name),
	}
	queryParameters := map[string]interface{}{}
	queryParameters["mount"] = autorest.Encode("query", reference)
	queryParameters["from"] = autorest.Encode("query", repoFrom)

	preparer := autorest.CreatePreparer(
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/v2/{name}/blobs/uploads/", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeader("authorization", client.Authorization))
	req, e := preparer.Prepare((&http.Request{}).WithContext(ctx))
	if e != nil {
		e = autorest.NewErrorWithError(e, "acrapi.BaseClient", "StartBlobUpload", nil, "Failure preparing request")
		return e
	}
	resp, e := client.StartBlobUploadSender(req)
	if e != nil {
		result.Response = autorest.Response{Response: resp}
		e = autorest.NewErrorWithError(e, "acrapi.BaseClient", "StartBlobUpload", resp, "Failure sending request")
		return e
	}

	result, e = client.StartBlobUploadResponder(resp)
	if e != nil {
		e = autorest.NewErrorWithError(e, "acrapi.BaseClient", "StartBlobUpload", resp, "Failure responding to request")
		return e
	}

	switch result.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
		var metadataError acrapi.Error
		if e = mapstructure.Decode(result, &metadataError); e == nil {
			return fmt.Errorf("%s %s", *(*metadataError.Errors)[0].Code, *(*metadataError.Errors)[0].Message)
		}
		return e
	default:
		return fmt.Errorf("unexpected response code: %v", result.StatusCode)
	}
}

// PutManifest creates a tag in a repository
func PutManifest(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string,
	manifest ManifestV2) error {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		"",
		"",
		auth,
		"",
		"",
		"",
		"")

	var uploadManifest acrapi.SetObject

	pathParameters := map[string]interface{}{
		"name":      autorest.Encode("path", client.Name),
		"reference": autorest.Encode("path", client.Reference),
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/v2/{name}/manifests/{reference}", pathParameters),
		autorest.WithHeader("Content-Type", "application/vnd.docker.distribution.manifest.v2+json"),
		autorest.WithHeader("authorization", client.Authorization))
	preparer = autorest.DecoratePreparer(preparer,
		autorest.WithJSON(manifest))
	req, e := preparer.Prepare((&http.Request{}).WithContext(ctx))

	if e != nil {
		e = autorest.NewErrorWithError(e, "acrapi.BaseClient", "UploadManifest", nil, "Failure preparing request")
		return e
	}
	resp, e := client.UploadManifestSender(req)
	if e != nil {
		uploadManifest.Response = autorest.Response{Response: resp}
		e = autorest.NewErrorWithError(e, "acrapi.BaseClient", "UploadManifest", resp, "Failure sending request")
		return e
	}

	uploadManifest, e = client.UploadManifestResponder(resp)
	if e != nil {
		e = autorest.NewErrorWithError(e, "acrapi.BaseClient", "UploadManifest", resp, "Failure responding to request")
		return e
	}

	switch uploadManifest.StatusCode {
	case http.StatusAccepted, http.StatusCreated:
		return nil
	case http.StatusBadRequest, http.StatusUnauthorized:
		var metadataError acrapi.Error
		if e = mapstructure.Decode(uploadManifest, &metadataError); e == nil {
			return fmt.Errorf("%s %s", *(*metadataError.Errors)[0].Code, *(*metadataError.Errors)[0].Message)
		}
		return errors.Wrap(e, "unable to decode error")
	default:
		return fmt.Errorf("unexpected response code: %v", uploadManifest.StatusCode)
	}
}

// AcrGetTagAttributes ...
func AcrGetTagAttributes(ctx context.Context,
	loginUrl string,
	auth string,
	repoName string,
	reference string) (*acrapi.TagAttributes, error) {
	hostname := GetHostname(loginUrl)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		reference,
		"",
		"",
		"",
		auth,
		"",
		"",
		"",
		"")

	if tagAttributes, e := client.AcrGetTagAttributes(ctx); e == nil {
		var acrGetTagAttributesResult acrapi.TagAttributes
		switch tagAttributes.StatusCode {
		case http.StatusOK:
			if e := mapstructure.Decode(tagAttributes.Value, &acrGetTagAttributesResult); e == nil {
				return &acrGetTagAttributesResult, nil
			}
			return nil, e
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var metadataError acrapi.Error
			if e := mapstructure.Decode(tagAttributes.Value, &metadataError); e == nil {
				return nil, fmt.Errorf("%s %s", *(*metadataError.Errors)[0].Code, *(*metadataError.Errors)[0].Message)
			} else {
				return nil, errors.Wrap(e, "unable to decode error")
			}
		default:
			return nil, fmt.Errorf("unexpected response code: %v", tagAttributes.StatusCode)
		}
	} else {
		return nil, e
	}
}

// AcrManifestMetadata the struct that is used to store original repository info
type AcrManifestMetadata struct {
	Digest         string    `json:"digest,omitempty"`
	OriginalRepo   string    `json:"repository,omitempty"`
	LastUpdateTime string    `json:"lastUpdateTime,omitempty"`
	Tags           []AcrTags `json:"tags,omitempty"`
}

// AcrTags stores the tag and the time it was archived
type AcrTags struct {
	Name        string `json:"name,omitempty"`
	ArchiveTime string `json:"archiveTime,omitempty"`
}

// ManifestV2 follows the docker manifest schema version 2
type ManifestV2 struct {
	SchemaVersion *int32           `json:"schemaVersion,omitempty"`
	MediaType     *string          `json:"mediaType,omitempty"`
	Config        *LayerMetadata   `json:"config,omitempty"`
	Layers        *[]LayerMetadata `json:"layers,omitempty"`
}

// LayerMetadata follows the schema for every layer in the docker manifest schema
type LayerMetadata struct {
	MediaType *string `json:"mediaType,omitempty"`
	Size      *int32  `json:"size,omitempty"`
	Digest    *string `json:"digest,omitempty"`
}
