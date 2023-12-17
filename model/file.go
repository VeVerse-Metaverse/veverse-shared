package model

import (
	"fmt"
	"github.com/gofrs/uuid"
)

type UploadImageType int64

//goland:noinspection GoUnusedConst
const (
	ImageFull UploadImageType = iota
	ImagePreview
	ImageTexture
)

// File trait for the Entity
type File struct {
	EntityTrait
	Timestamps
	Type         string     `json:"type"`
	Url          string     `json:"url"`
	Mime         *string    `json:"mime,omitempty"`
	Size         *int64     `json:"size,omitempty"`
	Version      int64      `json:"version,omitempty"`        // version of the file if versioned
	Deployment   string     `json:"deploymentType,omitempty"` // server or client if applicable
	Platform     string     `json:"platform,omitempty"`       // platform if applicable
	UploadedBy   *uuid.UUID `json:"uploadedBy,omitempty"`     // user that uploaded the file
	Width        *int       `json:"width,omitempty"`
	Height       *int       `json:"height,omitempty"`
	Index        int64      `json:"variation,omitempty"`    // variant of the file if applicable (e.g. PDF pages)
	OriginalPath *string    `json:"originalPath,omitempty"` // original relative path to maintain directory structure (e.g. for releases)
	Hash         *string    `json:"hash,omitempty"`
}

func (f *File) String() string {
	var out = f.EntityTrait.String()
	out += f.Timestamps.String()
	out += fmt.Sprintf("type: %v, ", f.Type)
	out += fmt.Sprintf("url: %v, ", f.Url)
	if f.Mime != nil {
		out += fmt.Sprintf("mime: %v, ", *f.Mime)
	}
	if f.Size != nil {
		out += fmt.Sprintf("size: %v, ", *f.Size)
	}
	out += fmt.Sprintf("version: %v, ", f.Version)
	out += fmt.Sprintf("deployment: %v, ", f.Deployment)
	out += fmt.Sprintf("platform: %v, ", f.Platform)
	if f.UploadedBy != nil {
		out += fmt.Sprintf("uploadedBy: %v, ", *f.UploadedBy)
	}
	if f.Width != nil {
		out += fmt.Sprintf("width: %v, ", *f.Width)
	}
	if f.Height != nil {
		out += fmt.Sprintf("height: %v, ", *f.Height)
	}
	out += fmt.Sprintf("index: %v, ", f.Index)
	if f.OriginalPath != nil {
		out += fmt.Sprintf("path: %v, ", *f.OriginalPath)
	}
	if f.Hash != nil {
		out += fmt.Sprintf("hash: %v, ", *f.Hash)
	}
	return out
}

type FileBatch Batch[File]

func (fb *FileBatch) String() string {
	var out = ""
	out += fmt.Sprintf("\"offset\": %v, ", fb.Offset)
	out += fmt.Sprintf("\"limit\": %v, ", fb.Limit)
	out += fmt.Sprintf("\"total\": %v, ", fb.Total)
	out += "["
	for _, f := range fb.Entities {
		out += "\n{" + f.String() + "},"
	}
	out += "\n]"
	return out
}

// FileBatchRequestMetadata Batch request metadata for requesting File entities
type FileBatchRequestMetadata struct {
	BatchRequestMetadata
	Type       string `json:"type,omitempty" query:"type"`             // SupportedPlatform (OS) of the pak file (Win64, Mac, Linux, IOS, Android)
	Platform   string `json:"platform,omitempty" query:"platform"`     // SupportedPlatform (OS) of the pak file (Win64, Mac, Linux, IOS, Android)
	Deployment string `json:"deployment,omitempty" query:"deployment"` // SupportedDeployment for the pak file (Server or Client)
}

type FileRequestMetadata struct {
	PackageRequestMetadata
}

type FileLinkRequestMetadata struct {
	Type         string  `json:"type,omitempty" query:"type"`                            // Type of the file
	Url          string  `json:"url,omitempty" query:"url"`                              // Url of the file
	Mime         *string `json:"mime,omitempty" query:"mime"`                            // Mime type (optional), by default set to binary/octet-stream)
	Size         *int    `json:"size,omitempty" query:"size"`                            // Size of the file (optional)
	Version      int64   `json:"version,omitempty" query:"version"`                      // Version of the file, automatically incremented if the file is re-uploaded or re-linked, used to check if the file has been updated and should be re-downloaded even if it has been cached
	Deployment   string  `json:"deployment,omitempty" query:"deployment"`                // Deployment for the destination pak file (Server or Client), usually set for package and release files
	Platform     string  `json:"platform,omitempty" query:"platform"`                    // Platform (OS) of the destination pak file (Win64, Mac, Linux, IOS, Android), usually set for package and release files
	Width        int     `json:"width,omitempty" query:"width"`                          // Width of the media surface (optional), usually set for multimedia files (images and videos)
	Height       int     `json:"height,omitempty" query:"height"`                        // Height of the media surface (optional), usually set for multimedia files (images and videos)
	Index        int64   `json:"index,omitempty" query:"index"`                          // Index of the file (for file arrays such as PDF pages rendered to images)
	OriginalPath string  `json:"originalPath,omitempty" query:"original-path,omitempty"` // Original path of the file (to be re-downloaded to the correct location, used by app release files)
}

type FileUploadRequestMetadata struct {
	Type         string  `json:"type,omitempty" query:"type"`                            // Type of the file
	Mime         *string `json:"mime,omitempty" query:"mime"`                            // Url of the file
	Version      int64   `json:"version,omitempty" query:"version"`                      // Version of the file, automatically incremented if the file is re-uploaded or re-linked, used to check if the file has been updated and should be re-downloaded even if it has been cached
	Deployment   string  `json:"deployment,omitempty" query:"deployment"`                // Deployment for the destination pak file (Server or Client), usually set for package and release files
	Platform     string  `json:"platform,omitempty" query:"platform"`                    // Platform (OS) of the destination pak file (Win64, Mac, Linux, IOS, Android), usually set for package and release files
	Width        int     `json:"width,omitempty" query:"width"`                          // Width of the media surface (optional), usually set for multimedia files (images and videos)
	Height       int     `json:"height,omitempty" query:"height"`                        // Height of the media surface (optional), usually set for multimedia files (images and videos)
	Index        int64   `json:"index,omitempty" query:"index"`                          // Index of the file (for file arrays such as PDF pages rendered to images)
	OriginalPath string  `json:"originalPath,omitempty" query:"original-path,omitempty"` // Original path of the file (to be re-downloaded to the correct location, used by app release files)
}

type FileUploadLinkRequestMetadata struct {
	FileId       *string `json:"fileId,omitempty"`                                       // SupportedPlatform (OS) of the pak file (Win64, Mac, Linux, IOS, Android)
	EntityId     string  `json:"entityId,omitempty"`                                     // SupportedPlatform (OS) of the pak file (Win64, Mac, Linux, IOS, Android)
	Type         string  `json:"type,omitempty" query:"type"`                            // Type of the file
	Url          *string `json:"url,omitempty" query:"url"`                              // Url of the file
	Mime         *string `json:"mime,omitempty" query:"mime"`                            // Mime type (optional), by default set to binary/octet-stream)
	Size         *int    `json:"size,omitempty" query:"size"`                            // Size of the file (optional)
	Version      int64   `json:"version,omitempty" query:"version"`                      // Version of the file, automatically incremented if the file is re-uploaded or re-linked, used to check if the file has been updated and should be re-downloaded even if it has been cached
	Deployment   string  `json:"deployment,omitempty" query:"deployment"`                // Deployment for the destination pak file (Server or Client), usually set for package and release files
	Platform     string  `json:"platform,omitempty" query:"platform"`                    // Platform (OS) of the destination pak file (Win64, Mac, Linux, IOS, Android), usually set for package and release files
	Width        int     `json:"width,omitempty" query:"width"`                          // Width of the media surface (optional), usually set for multimedia files (images and videos)
	Height       int     `json:"height,omitempty" query:"height"`                        // Height of the media surface (optional), usually set for multimedia files (images and videos)
	Index        int64   `json:"index,omitempty" query:"index"`                          // Index of the file (for file arrays such as PDF pages rendered to images)
	OriginalPath string  `json:"originalPath,omitempty" query:"original-path,omitempty"` // Original path of the file (to be re-downloaded to the correct location, used by app release files)
}

type FileDownloadRequestMetadata struct {
	EntityId string `json:"entityId,omitempty"` // SupportedPlatform (OS) of the pak file (Win64, Mac, Linux, IOS, Android)
	FileId   string `json:"fileId,omitempty"`   // SupportedPlatform (OS) of the pak file (Win64, Mac, Linux, IOS, Android)
}
