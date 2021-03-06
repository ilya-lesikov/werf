package file_reader

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/werf/werf/pkg/giterminism_manager/errors"
)

type FilesNotFoundInTheProjectDirectoryError struct {
	error
}
type FilesNotFoundInTheProjectGitRepositoryError struct {
	error
}
type UncommittedFilesError struct {
	error
}
type UncommittedFilesChangesError struct {
	error
}

func isUncommittedFilesChangesError(err error) bool {
	switch err.(type) {
	case UncommittedFilesChangesError:
		return true
	default:
		return false
	}
}

func NewFilesNotFoundInTheProjectDirectoryError(configType configType, relPaths ...string) error {
	var errorMsg string
	if len(relPaths) == 1 {
		errorMsg = fmt.Sprintf("the %s %q not found in the project git repository", configType, filepath.ToSlash(relPaths[0]))
	} else if len(relPaths) > 1 {
		errorMsg = fmt.Sprintf("the following %ss not found in the project git repository:\n\n%s", configType, prepareListOfFilesString(relPaths))
	} else {
		panic("unexpected condition")
	}

	return FilesNotFoundInTheProjectDirectoryError{errors.NewError(errorMsg)}
}

func NewFilesNotFoundInTheProjectGitRepositoryError(configType configType, relPaths ...string) error {
	var errorMsg string
	if len(relPaths) == 1 {
		errorMsg = fmt.Sprintf("the %s %q not found in the project git repository", configType, filepath.ToSlash(relPaths[0]))
	} else if len(relPaths) > 1 {
		errorMsg = fmt.Sprintf("the following %ss not found in the project git repository:\n\n%s", configType, prepareListOfFilesString(relPaths))
	} else {
		panic("unexpected condition")
	}

	return FilesNotFoundInTheProjectGitRepositoryError{errors.NewError(errorMsg)}
}

func NewUncommittedFilesError(configType configType, relPaths ...string) error {
	errorMsg := "the uncommitted configuration found in the project git work tree"
	if len(relPaths) == 1 {
		errorMsg = fmt.Sprintf("%s: the %s %q must be committed", errorMsg, configType, filepath.ToSlash(relPaths[0]))
	} else if len(relPaths) > 1 {
		errorMsg = fmt.Sprintf("%s: the following %ss must be committed:\n\n%s", errorMsg, configType, prepareListOfFilesString(relPaths))
	} else {
		panic("unexpected condition")
	}

	return UncommittedFilesError{errors.NewError(errorMsg)}
}

func NewUncommittedFilesChangesError(configType configType, relPaths ...string) error {
	errorMsg := "the uncommitted configuration found in the project git work tree"
	if len(relPaths) == 1 {
		errorMsg = fmt.Sprintf("%s: the %s %q changes must be committed", errorMsg, configType, filepath.ToSlash(relPaths[0]))
	} else if len(relPaths) > 1 {
		errorMsg = fmt.Sprintf("%s: the following %ss changes must be committed:\n\n%s", errorMsg, configType, prepareListOfFilesString(relPaths))
	} else {
		panic("unexpected condition")
	}

	return UncommittedFilesChangesError{errors.NewError(errorMsg)}
}

func prepareListOfFilesString(paths []string) string {
	var result string
	for _, path := range paths {
		result += " - " + filepath.ToSlash(path) + "\n"
	}

	return strings.TrimSuffix(result, "\n")
}
