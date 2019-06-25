// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	acrapi "github.com/AzureCR/acr-cli/acr"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

const (
	prefixHTTPS = "https://"
	registryURL = ".azurecr.io"
)

// BasicAuth returns the username and the passwrod encoded in base 64.
func BasicAuth(username string, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// LoginURL returns the FQDN for a registry.
func LoginURL(registryName string) string {
	// TODO: if the registry is in another cloud (i.err. dogfood) a full FQDN for the registry should be specified.
	if strings.Contains(registryName, ".") {
		return registryName
	}
	return registryName + registryURL
}

// LoginURLWithPrefix return the hostname of a registry.
func LoginURLWithPrefix(loginURL string) string {
	urlWithPrefix := loginURL
	if !strings.HasPrefix(loginURL, prefixHTTPS) {
		urlWithPrefix = prefixHTTPS + loginURL
	}
	return urlWithPrefix
}

// AcrListTags list the tags of a repository with their attributes.
func AcrListTags(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	orderBy string,
	last string) (*acrapi.TagAttributeList, error) {
	hostname := LoginURLWithPrefix(loginURL)
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
	tags, err := client.AcrListTags(ctx)
	if err != nil {
		return nil, err
	}
	var listTagResult acrapi.TagAttributeList
	switch tags.StatusCode {
	case http.StatusOK:
		if err = mapstructure.Decode(tags.Value, &listTagResult); err != nil {
			return nil, err
		}
		return &listTagResult, nil

	case http.StatusUnauthorized, http.StatusNotFound:
		var apiError acrapi.Error
		if err = mapstructure.Decode(tags.Value, &apiError); err != nil {
			return nil, errors.Wrap(err, "unable to decode error")
		}
		if apiError.Errors != nil && len(*apiError.Errors) > 0 {
			return nil, fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
		}
		return nil, errors.New("unable to decode apiError")

	default:
		return nil, fmt.Errorf("unexpected response code: %v", tags.StatusCode)
	}
}

// AcrDeleteTag deletes the tag by reference.
func AcrDeleteTag(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string) error {
	hostname := LoginURLWithPrefix(loginURL)
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
	tag, err := client.AcrDeleteTag(ctx)
	if err != nil {
		return err
	}
	switch tag.StatusCode {
	case http.StatusAccepted:
		return nil
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
		var apiError acrapi.Error
		if err = mapstructure.Decode(tag, &apiError); err != nil {
			return errors.Wrap(err, "unable to decode error")
		}
		if apiError.Errors != nil && len(*apiError.Errors) > 0 {
			return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
		}
		return errors.New("unable to decode apiError")

	default:
		return fmt.Errorf("unexpected response code: %v", tag.StatusCode)
	}
}

// AcrListManifests list all the manifest in a repository with their attributes.
func AcrListManifests(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	orderBy string,
	last string) (*acrapi.ManifestAttributeList, error) {
	hostname := LoginURLWithPrefix(loginURL)
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
	manifests, err := client.AcrListManifests(ctx)
	if err != nil {
		return nil, err
	}
	switch manifests.StatusCode {
	case http.StatusOK:
		var acrListManifestsAttributesResult acrapi.ManifestAttributeList
		if err = mapstructure.Decode(manifests.Value, &acrListManifestsAttributesResult); err != nil {
			return nil, err
		}
		return &acrListManifestsAttributesResult, nil

	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
		var apiError acrapi.Error
		if err = mapstructure.Decode(manifests.Value, &apiError); err != nil {
			return nil, errors.Wrap(err, "unable to decode error")
		}
		if apiError.Errors != nil && len(*apiError.Errors) > 0 {
			return nil, fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
		}
		return nil, errors.New("unable to decode apiError")

	default:
		return nil, fmt.Errorf("unexpected response code: %v", manifests.StatusCode)
	}
}

// DeleteManifest deletes a manifest using the digest as a reference.
func DeleteManifest(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string) error {
	hostname := LoginURLWithPrefix(loginURL)
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
	deleteManifest, err := client.DeleteManifest(ctx)
	if err != nil {
		return err
	}
	switch deleteManifest.StatusCode {
	case http.StatusAccepted:
		return nil

	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
		var apiError acrapi.Error
		if err = mapstructure.Decode(deleteManifest, &apiError); err != nil {
			return errors.Wrap(err, "unable to decode error")
		}
		if apiError.Errors != nil && len(*apiError.Errors) > 0 {
			return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
		}
		return errors.New("unable to decode apiError")

	default:
		return fmt.Errorf("unexpected response code: %v", deleteManifest.StatusCode)
	}
}

// AcrGetManifestMetadata get the metadata of a manifest
func AcrGetManifestMetadata(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string,
	metadataName string) (*string, error) {
	hostname := LoginURLWithPrefix(loginURL)
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
	tagMetadata, err := client.AcrGetManifestMetadata(ctx)
	if err != nil {
		return nil, err
	}
	var acrGetTagMetadataResult string
	switch tagMetadata.StatusCode {
	case http.StatusOK:
		if err = mapstructure.Decode(tagMetadata.Value, &acrGetTagMetadataResult); err != nil {
			return nil, err
		}
		return &acrGetTagMetadataResult, nil

	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
		var apiError acrapi.Error
		if err = mapstructure.Decode(tagMetadata.Value, &apiError); err != nil {
			return nil, errors.Wrap(err, "unable to decode error")
		}
		if apiError.Errors != nil && len(*apiError.Errors) > 0 {
			return nil, fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
		}
		return nil, errors.New("unable to decode apiError")

	default:
		return nil, fmt.Errorf("unexpected response code: %v", tagMetadata.StatusCode)
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
	hostname := LoginURLWithPrefix(loginURL)
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
	manifestMetadata, err := client.AcrUpdateManifestMetadata(ctx, value)
	if err != nil {
		return err
	}
	switch manifestMetadata.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
		var apiError acrapi.Error
		if err = mapstructure.Decode(manifestMetadata, &apiError); err != nil {
			return err
		}
		if apiError.Errors != nil && len(*apiError.Errors) > 0 {
			return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
		}
		return errors.New("unable to decode apiError")

	default:
		return fmt.Errorf("unexpected response code: %v", manifestMetadata.StatusCode)
	}
}

// GetManifest returns the V2 manifest schema
func GetManifest(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string) (*string, error) {
	hostname := LoginURLWithPrefix(loginURL)
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

	var manifest acrapi.SetObject

	req, err := client.GetManifestPreparer(ctx)
	if err != nil {
		err = autorest.NewErrorWithError(err, "acrapi.BaseClient", "GetManifest", nil, "Failure preparing request")
		return nil, err
	}

	resp, err := client.GetManifestSender(req)
	if err != nil {
		manifest.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "acrapi.BaseClient", "GetManifest", resp, "Failure sending request")
		return nil, err
	}

	manifestBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	manifestString := string(manifestBytes)

	return &manifestString, nil
}

// AcrCrossReferenceLayer ...
func AcrCrossReferenceLayer(ctx context.Context,
	loginURL string,
	auth string,
	repoName string,
	reference string,
	repoFrom string) error {
	hostname := LoginURLWithPrefix(loginURL)
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
	req, err := preparer.Prepare((&http.Request{}).WithContext(ctx))
	if err != nil {
		err = autorest.NewErrorWithError(err, "acrapi.BaseClient", "StartBlobUpload", nil, "Failure preparing request")
		return err
	}
	resp, err := client.StartBlobUploadSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "acrapi.BaseClient", "StartBlobUpload", resp, "Failure sending request")
		return err
	}

	result, err = client.StartBlobUploadResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "acrapi.BaseClient", "StartBlobUpload", resp, "Failure responding to request")
		return err
	}

	switch result.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
		var apiError acrapi.Error
		if err = mapstructure.Decode(result.Value, &apiError); err != nil {
			return errors.Wrap(err, "unable to decode error")
		}
		if apiError.Errors != nil && len(*apiError.Errors) > 0 {
			return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
		}
		return errors.New("unable to decode apiError")

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
	manifest string) error {
	hostname := LoginURLWithPrefix(loginURL)
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
		autorest.WithString(manifest),
		autorest.WithHeader("authorization", client.Authorization))
	req, err := preparer.Prepare((&http.Request{}).WithContext(ctx))
	if err != nil {
		err = autorest.NewErrorWithError(err, "acrapi.BaseClient", "UploadManifest", nil, "Failure preparing request")
		return err
	}

	resp, err := client.UploadManifestSender(req)
	if err != nil {
		uploadManifest.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "acrapi.BaseClient", "UploadManifest", resp, "Failure sending request")
		return err
	}

	uploadManifest, err = client.UploadManifestResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "acrapi.BaseClient", "UploadManifest", resp, "Failure responding to request")
		return err
	}

	switch uploadManifest.StatusCode {
	case http.StatusAccepted, http.StatusCreated:
		return nil
	case http.StatusBadRequest, http.StatusUnauthorized:
		var apiError acrapi.Error
		if err = mapstructure.Decode(uploadManifest.Value, &apiError); err != nil {
			return errors.Wrap(err, "unable to decode error")
		}
		if apiError.Errors != nil && len(*apiError.Errors) > 0 {
			return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
		}
		return errors.New("unable to decode apiError")

	default:
		return fmt.Errorf("unexpected response code: %v", uploadManifest.StatusCode)
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
	hostname := LoginURLWithPrefix(loginURL)
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
	tagMetadata, err := client.AcrUpdateTagMetadata(ctx, value)
	if err != nil {
		return err
	}
	switch tagMetadata.StatusCode {
	case http.StatusCreated:
		return nil

	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
		var apiError acrapi.Error
		if err = mapstructure.Decode(tagMetadata.Value, &apiError); err != nil {
			return errors.Wrap(err, "unable to decode error")
		}
		if apiError.Errors != nil && len(*apiError.Errors) > 0 {
			return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
		}
		return errors.New("unable to decode apiError")

	default:
		return fmt.Errorf("unexpected response code: %v", tagMetadata.StatusCode)
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
