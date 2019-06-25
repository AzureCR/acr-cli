package acrapi

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
    "context"
    "users/t-gigo/go/src/github.com/azurecr/acr-cli/acr"
    "github.com/Azure/go-autorest/autorest"
)

        // BaseClientAPI contains the set of methods on the BaseClient type.
        type BaseClientAPI interface {
            CancelBlobUpload(ctx context.Context, UUID string) (result acr.SetObject, err error)
            CheckBlobExistence(ctx context.Context) (result autorest.Response, err error)
            CheckDockerRegistryV2Support(ctx context.Context) (result acr.CheckDockerRegistryV2SupportUnauthorizedResponse, err error)
            DeleteManifest(ctx context.Context) (result acr.SetObject, err error)
            DeleteManifestMetadata(ctx context.Context) (result acr.SetObject, err error)
            DeleteRepository(ctx context.Context) (result acr.SetObject, err error)
            DeleteRepositoryMetadata(ctx context.Context) (result acr.SetObject, err error)
            DeleteTag(ctx context.Context) (result acr.SetObject, err error)
            DeleteTagMetadata(ctx context.Context) (result acr.SetObject, err error)
            EndBlobUpload(ctx context.Context, digest string, UUID string) (result acr.SetObject, err error)
            GetBlob(ctx context.Context) (result acr.SetObject, err error)
            GetBlobUploadStatus(ctx context.Context, UUID string) (result acr.SetObject, err error)
            GetManifest(ctx context.Context) (result acr.SetObject, err error)
            GetManifestAttributes(ctx context.Context) (result acr.SetObject, err error)
            GetManifestMetadata(ctx context.Context) (result acr.SetObject, err error)
            GetRepositoryAttributes(ctx context.Context) (result acr.SetObject, err error)
            GetRepositoryMetadata(ctx context.Context) (result acr.SetObject, err error)
            GetTagAttributes(ctx context.Context) (result acr.SetObject, err error)
            GetTagMetadata(ctx context.Context) (result acr.SetObject, err error)
            ListManifestMetadata(ctx context.Context) (result acr.SetObject, err error)
            ListManifests(ctx context.Context) (result acr.SetObject, err error)
            ListRepositories(ctx context.Context) (result acr.SetObject, err error)
            ListRepositoriesMethod(ctx context.Context) (result acr.SetObject, err error)
            ListRepositoryMetadata(ctx context.Context) (result acr.SetObject, err error)
            ListTagMetadata(ctx context.Context) (result acr.SetObject, err error)
            ListTags(ctx context.Context) (result acr.SetObject, err error)
            ListTagsMethod(ctx context.Context) (result acr.SetObject, err error)
            StartBlobUpload(ctx context.Context, digest string) (result acr.SetObject, err error)
            UpdateManifestAttributes(ctx context.Context, value string) (result acr.SetObject, err error)
            UpdateManifestMetadata(ctx context.Context, value string) (result acr.SetObject, err error)
            UpdateRepositoryAttributes(ctx context.Context, value string) (result acr.SetObject, err error)
            UpdateRepositoryMetadata(ctx context.Context, value string) (result acr.SetObject, err error)
            UpdateTagAttributes(ctx context.Context, value string) (result acr.SetObject, err error)
            UpdateTagMetadata(ctx context.Context, value string) (result acr.SetObject, err error)
            UploadBlobContent(ctx context.Context, UUID string) (result acr.SetObject, err error)
            UploadManifest(ctx context.Context) (result acr.SetObject, err error)
        }

        var _ BaseClientAPI = (*acr.BaseClient)(nil)
