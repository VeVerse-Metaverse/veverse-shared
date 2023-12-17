package helper

import (
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/karrick/godirwalk"
	"path/filepath"
	"strings"
)

func ToSliceOfAny[T any](s []T) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

func SanitizeLikeClause(s string) string {
	return strings.ReplaceAll(s, "%", "\\%")
}

func DescribeRows(rows pgx.Rows) (out string) {
	desc := rows.FieldDescriptions()
	values, _ := rows.Values()

	for i, v := range values {
		out += fmt.Sprintf("{\"%v\":\"%v\"}\n", desc[i].Name, v)
	}

	return
}

func ListFilesRecursive(root string, ignore []string) ([]string, error) {
	var files []string
	err := godirwalk.Walk(root, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			relPath, _ := filepath.Rel(root, path)
			for _, ignoredFile := range ignore {
				if strings.HasPrefix(relPath, ignoredFile) || strings.HasPrefix(filepath.Base(path), ignoredFile) {
					return nil
				}
			}

			if !de.IsDir() {
				files = append(files, relPath)
			}
			return nil
		},
		Unsorted: true,
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}
