package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	acrapi "github.com/Azure/libacr/golang"
	"github.com/mitchellh/mapstructure"
)

// The constants used in the sdk.
const (
	OrderByTimeAsc    = "timeasc"
	OrderByTimeDesc   = "timedesc"
	MaxEntries        = 1000
	HTTPSPrefix       = "https://"
	MediaTypeManifest = "application/vnd.docker.distribution.manifest.v2+json"
)

var errParse = errors.New("Error parsing")
var errResponseCode = errors.New("Undefined response code")

// BasicAuth returns the username and the passwrod encoded in base 64
func BasicAuth(username string,
	password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// ListTags list all the tags associated to a repository
func ListTags(loginURL string,
	auth string,
	repoName string) (*Tags, error) {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		"",
		"",
		"",
		"",
		auth,
		"",
		"1000",
		"",
		"")
	if tags, err := client.ListTags(context.Background()); err == nil {
		var listTagResult Tags
		switch tags.StatusCode {
		case http.StatusOK:
			if err := mapstructure.Decode(tags.Value, &listTagResult); err == nil {
				return &listTagResult, nil
			}
			return nil, errParse

		case http.StatusUnauthorized, http.StatusNotFound:
			var apiError acrapi.Error
			if err := mapstructure.Decode(tags.Value, &apiError); err == nil {
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

// AcrGetTagAttributes gets the attributes of a tag.
func AcrGetTagAttributes(loginURL string,
	auth string,
	repoName string,
	reference string) (*acrapi.TagAttributes, error) {
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
	if tagAttributes, err := client.AcrGetTagAttributes(context.Background()); err == nil {
		var acrGetTagAttributes acrapi.TagAttributes
		switch tagAttributes.StatusCode {
		case http.StatusOK:
			if err := mapstructure.Decode(tagAttributes.Value, &acrGetTagAttributes); err == nil {
				return &acrGetTagAttributes, nil
			}
			return nil, errParse

		case http.StatusUnauthorized, http.StatusNotFound:
			var apiError acrapi.Error
			if err := mapstructure.Decode(tagAttributes.Value, &apiError); err == nil {
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
func AcrDeleteTag(loginURL string,
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

	if tag, err := client.AcrDeleteTag(context.Background()); err == nil {
		switch tag.StatusCode {
		case http.StatusAccepted:
			return nil
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if err := mapstructure.Decode(tag, &apiError); err == nil {
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
func AcrListManifests(loginURL string,
	auth string,
	repoName string,
	orderBy string,
	last string,
	maxEntries int) (*acrapi.ManifestAttributeList, error) {
	hostname := GetHostname(loginURL)
	client := acrapi.NewWithBaseURI(hostname,
		repoName,
		"",
		"",
		"",
		"",
		auth,
		orderBy,
		strconv.Itoa(maxEntries),
		last,
		"")

	if manifests, err := client.AcrListManifests(context.Background()); err == nil {
		switch manifests.StatusCode {
		case http.StatusOK:
			var acrListManifestsAttributesResult acrapi.ManifestAttributeList
			if err := mapstructure.Decode(manifests.Value, &acrListManifestsAttributesResult); err == nil {
				return &acrListManifestsAttributesResult, nil
			}
			return nil, errParse

		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if err := mapstructure.Decode(manifests.Value, &apiError); err == nil {
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

// GetHostname return the hostname of a registry
func GetHostname(loginURL string) string {
	hostname := loginURL
	if !strings.HasPrefix(loginURL, HTTPSPrefix) {
		hostname = HTTPSPrefix + loginURL
	}

	return hostname
}

// DeleteManifest deletes a manifest using the digest as a reference.
func DeleteManifest(loginURL string,
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

	if deleteManifest, err := client.DeleteManifest(context.Background()); err == nil {
		switch deleteManifest.StatusCode {
		case http.StatusAccepted:
			return nil
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusMethodNotAllowed:
			var apiError acrapi.Error
			if err := mapstructure.Decode(deleteManifest, &apiError); err == nil {
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

// Tags is a struct used for decoding the response of client.ListTags
type Tags struct {
	Name *string   `json:"name,omitempty"`
	Tags *[]string `json:"tags,omitempty"`
}
