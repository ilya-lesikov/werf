package context_manager

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	uuid "github.com/satori/go.uuid"

	"github.com/werf/logboek"

	"github.com/werf/werf/pkg/path_matcher"
	"github.com/werf/werf/pkg/util"
	"github.com/werf/werf/pkg/werf"
)

func GetTmpDir() string {
	return filepath.Join(werf.GetServiceDir(), "tmp", "context")
}

func GetTmpArchivePath() string {
	return filepath.Join(GetTmpDir(), uuid.NewV4().String())
}

func ContextAddFileChecksum(ctx context.Context, projectDir string, contextDir string, contextAddFile []string, matcher path_matcher.PathMatcher) (string, error) {
	var filePathListRelativeToProject []string
	for _, addFileRelativeToContext := range contextAddFile {
		addFileRelativeToProject := filepath.Join(contextDir, addFileRelativeToContext)
		if !matcher.MatchPath(addFileRelativeToProject) {
			continue
		}

		filePathListRelativeToProject = append(filePathListRelativeToProject, addFileRelativeToProject)
	}

	if len(filePathListRelativeToProject) == 0 {
		return "", nil
	}

	h := sha256.New()
	for _, pathRelativeToProject := range filePathListRelativeToProject {
		pathWithSlashes := filepath.ToSlash(pathRelativeToProject)
		h.Write([]byte(pathWithSlashes))

		absolutePath := filepath.Join(projectDir, pathRelativeToProject)
		if exists, err := util.RegularFileExists(absolutePath); err != nil {
			return "", fmt.Errorf("unable to check existence of file %q: %s", absolutePath, err)
		} else if !exists {
			continue
		}

		if err := func() error {
			f, err := os.Open(absolutePath)
			if err != nil {
				return fmt.Errorf("unable to open %q: %s", absolutePath, err)
			}
			defer f.Close()

			if _, err := io.Copy(h, f); err != nil {
				return fmt.Errorf("unable to copy file %q: %s", absolutePath, err)
			}

			return nil
		}(); err != nil {
			return "", err
		}

		logboek.Context(ctx).Debug().LogF("File was added: %q\n", pathWithSlashes)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func AddContextAddFileToContextArchive(ctx context.Context, originalArchivePath string, projectDir string, contextDir string, contextAddFile []string) (string, error) {
	destinationArchivePath := GetTmpArchivePath()

	pathsToExcludeFromSourceArchive := contextAddFile
	if err := util.CreateArchiveBasedOnAnotherOne(ctx, originalArchivePath, destinationArchivePath, pathsToExcludeFromSourceArchive, func(tw *tar.Writer) error {
		var filesToCopy []string
		for _, contextAddFile := range contextAddFile {
			contextAddFilePath := filepath.Join(projectDir, contextDir, contextAddFile)

			contextAddFileInfo, err := os.Lstat(contextAddFilePath)
			if err != nil {
				return fmt.Errorf("unable to get file info for contextAddFile %q: %s", contextAddFilePath, err)
			}

			if contextAddFileInfo.IsDir() {
				if err := filepath.Walk(contextAddFilePath, func(path string, fileInfo os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if fileInfo.IsDir() {
						return nil
					}
					filesToCopy = append(filesToCopy, path)
					logboek.Context(ctx).Debug().LogF("Extra file is going to be added to the current context %q\n", path)
					return nil
				}); err != nil {
					return fmt.Errorf("error occured when recursively walking the contextAddFile dir %q: %s", contextAddFilePath, err)
				}
			} else {
				filesToCopy = append(filesToCopy, contextAddFilePath)
				logboek.Context(ctx).Debug().LogF("Extra file is going to be added to the current context: %q\n", contextAddFilePath)
			}
		}

		for _, fileToCopy := range filesToCopy {
			tarEntryName, err := filepath.Rel(filepath.Join(projectDir, contextDir), fileToCopy)
			if err != nil {
				return fmt.Errorf("unable to get context relative path for %q: %s", fileToCopy, err)
			}
			tarEntryName = filepath.ToSlash(tarEntryName)
			if err := util.CopyFileIntoTar(tw, tarEntryName, fileToCopy); err != nil {
				return fmt.Errorf("unable to add contextAddFile %q to archive %q: %s", fileToCopy, destinationArchivePath, err)
			}
			logboek.Context(ctx).Debug().LogF("Extra file was added to the current context: %q\n", tarEntryName)
		}

		return nil
	}); err != nil {
		return "", err
	}

	return destinationArchivePath, nil
}
