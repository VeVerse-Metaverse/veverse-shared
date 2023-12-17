package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func addToZip(zipWriter *zip.Writer, basePath, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(basePath, path)
	if err != nil {
		return err
	}

	header.Name = filepath.ToSlash(relPath)
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

func CreateZipArchive(output, basePath string, files []string) error {
	zipFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			fmt.Printf("failed to close zip file: %v", err)
		}
	}(zipFile)

	zipWriter := zip.NewWriter(zipFile)
	defer func(zipWriter *zip.Writer) {
		err := zipWriter.Close()
		if err != nil {
			fmt.Printf("failed to close zip writer: %v", err)
		}
	}(zipWriter)

	for _, file := range files {
		filePath := filepath.Join(basePath, file)
		err = addToZip(zipWriter, basePath, filePath)
		if err != nil {
			return fmt.Errorf("failed to add file %s to zip: %v", filePath, err)
		}
	}

	return nil
}
