// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	acrapi "github.com/Azure/libacr/golang"
	"github.com/mitchellh/mapstructure"
)

const (
	prefixHTTPS = "https://"
	registryURL = ".azurecr.io"
)

var errParse = errors.New("error parsing")
var errResponseCode = errors.New("undefined response code")

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
	if tags, err := client.AcrListTags(ctx); err == nil {
		var listTagResult acrapi.TagAttributeList
		switch tags.StatusCode {
		case http.StatusOK:
			if err = mapstructure.Decode(tags.Value, &listTagResult); err == nil {
				return &listTagResult, nil
			}
			return nil, errParse

		case http.StatusUnauthorized, http.StatusNotFound:
			var apiError acrapi.Error
			if err = mapstructure.Decode(tags.Value, &apiError); err == nil {
				return nil, fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return nil, errParse

		default:
			return nil, errResponseCode
		}
	} else {
		return nil, err
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

	if tag, err := client.AcrDeleteTag(ctx); err == nil {
		switch tag.StatusCode {
		case http.StatusAccepted:
			return nil
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if err = mapstructure.Decode(tag, &apiError); err == nil {
				return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return errParse

		default:
			return errResponseCode
		}
	} else {
		return err
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

	if manifests, err := client.AcrListManifests(ctx); err == nil {
		switch manifests.StatusCode {
		case http.StatusOK:
			var acrListManifestsAttributesResult acrapi.ManifestAttributeList
			if err = mapstructure.Decode(manifests.Value, &acrListManifestsAttributesResult); err == nil {
				return &acrListManifestsAttributesResult, nil
			}
			return nil, errParse

		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if err = mapstructure.Decode(manifests.Value, &apiError); err == nil {
				return nil, fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return nil, errParse

		default:
			return nil, errResponseCode
		}
	} else {
		return nil, err
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

	if deleteManifest, err := client.DeleteManifest(ctx); err == nil {
		switch deleteManifest.StatusCode {
		case http.StatusAccepted:
			return nil
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if err = mapstructure.Decode(deleteManifest, &apiError); err == nil {
				return fmt.Errorf("%s %s", *(*apiError.Errors)[0].Code, *(*apiError.Errors)[0].Message)
			}
			return errParse

		default:
			return errResponseCode
		}
	} else {
		return err
	}
}
