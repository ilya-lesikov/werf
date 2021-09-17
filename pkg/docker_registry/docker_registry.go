package docker_registry

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/werf/werf/pkg/image"
)

type DockerRegistry interface {
	CreateRepo(ctx context.Context, reference string) error
	DeleteRepo(ctx context.Context, reference string) error
	Tags(ctx context.Context, reference string) ([]string, error)
	CheckRepoImageCustomTag(ctx context.Context, repoImage *image.Info, tag string) error
	TagRepoImage(ctx context.Context, repoImage *image.Info, tag string) error
	GetRepoImage(ctx context.Context, reference string) (*image.Info, error)
	TryGetRepoImage(ctx context.Context, reference string) (*image.Info, error)
	IsRepoImageExists(ctx context.Context, reference string) (bool, error)
	DeleteRepoImage(ctx context.Context, repoImage *image.Info) error
	PushImage(ctx context.Context, reference string, opts *PushImageOptions) error
	MutateAndPushImage(ctx context.Context, sourceReference, destinationReference string, mutateConfigFunc func(v1.Config) (v1.Config, error)) error

	String() string
}

type PushImageOptions struct {
	Labels map[string]string
}

type DockerRegistryOptions struct {
	InsecureRegistry      bool
	SkipTlsVerifyRegistry bool
	DockerHubToken        string
	DockerHubUsername     string
	DockerHubPassword     string
	GitHubToken           string
	HarborUsername        string
	HarborPassword        string
	QuayToken             string
}

func (o *DockerRegistryOptions) awsEcrOptions() awsEcrOptions {
	return awsEcrOptions{
		defaultImplementationOptions: o.defaultOptions(),
	}
}

func (o *DockerRegistryOptions) azureAcrOptions() azureCrOptions {
	return azureCrOptions{
		defaultImplementationOptions: o.defaultOptions(),
	}
}

func (o *DockerRegistryOptions) dockerHubOptions() dockerHubOptions {
	return dockerHubOptions{
		defaultImplementationOptions: o.defaultOptions(),
		dockerHubCredentials: dockerHubCredentials{
			token:    o.DockerHubToken,
			username: o.DockerHubUsername,
			password: o.DockerHubPassword,
		},
	}
}

func (o *DockerRegistryOptions) gcrOptions() GcrOptions {
	return GcrOptions{
		defaultImplementationOptions: o.defaultOptions(),
	}
}

func (o *DockerRegistryOptions) gitHubPackagesOptions() gitHubPackagesOptions {
	return gitHubPackagesOptions{
		defaultImplementationOptions: o.defaultOptions(),
		gitHubCredentials: gitHubCredentials{
			token: o.GitHubToken,
		},
	}
}

func (o *DockerRegistryOptions) gitLabRegistryOptions() gitLabRegistryOptions {
	return gitLabRegistryOptions{
		defaultImplementationOptions: o.defaultOptions(),
	}
}

func (o *DockerRegistryOptions) harborOptions() harborOptions {
	return harborOptions{
		defaultImplementationOptions: o.defaultOptions(),
		harborCredentials: harborCredentials{
			username: o.HarborUsername,
			password: o.HarborPassword,
		},
	}
}

func (o *DockerRegistryOptions) quayOptions() quayOptions {
	return quayOptions{
		defaultImplementationOptions: o.defaultOptions(),
		quayCredentials: quayCredentials{
			token: o.QuayToken,
		},
	}
}

func (o *DockerRegistryOptions) defaultOptions() defaultImplementationOptions {
	return defaultImplementationOptions{apiOptions{
		InsecureRegistry:      o.InsecureRegistry,
		SkipTlsVerifyRegistry: o.SkipTlsVerifyRegistry,
	}}
}

func NewDockerRegistry(repositoryAddress string, implementation string, options DockerRegistryOptions) (DockerRegistry, error) {
	switch implementation {
	case AwsEcrImplementationName:
		return newAwsEcr(options.awsEcrOptions())
	case AzureCrImplementationName:
		return newAzureCr(options.azureAcrOptions())
	case DockerHubImplementationName:
		return newDockerHub(options.dockerHubOptions())
	case GcrImplementationName:
		return newGcr(options.gcrOptions())
	case GitHubPackagesImplementationName:
		return newGitHubPackages(options.gitHubPackagesOptions())
	case GitLabRegistryImplementationName:
		return newGitLabRegistry(options.gitLabRegistryOptions())
	case HarborImplementationName:
		return newHarbor(options.harborOptions())
	case QuayImplementationName:
		return newQuay(options.quayOptions())
	case DefaultImplementationName:
		return newDefaultImplementation(options.defaultOptions())
	default:
		resolvedImplementation, err := ResolveImplementation(repositoryAddress, implementation)
		if err != nil {
			return nil, err
		}

		return NewDockerRegistry(repositoryAddress, resolvedImplementation, options)
	}
}

func ResolveImplementation(repository, implementation string) (string, error) {
	for _, supportedImplementation := range ImplementationList() {
		if supportedImplementation == implementation {
			return implementation, nil
		}
	}

	if implementation == "auto" || implementation == "" {
		return detectImplementation(repository)
	}

	return "", fmt.Errorf("docker registry implementation %s is not supported", implementation)
}

func detectImplementation(accountOrRepositoryAddress string) (string, error) {
	var parsedResource authn.Resource
	var err error

	parts := strings.SplitN(accountOrRepositoryAddress, "/", 2)
	if len(parts) == 1 && (strings.ContainsRune(parts[0], '.') || strings.ContainsRune(parts[0], ':')) {
		parsedResource, err = name.NewRegistry(accountOrRepositoryAddress)
		if err != nil {
			return "", err
		}
	} else {
		parsedResource, err = name.NewRepository(accountOrRepositoryAddress)
		if err != nil {
			return "", err
		}
	}

	for _, service := range []struct {
		name     string
		patterns []string
	}{
		{
			name:     AwsEcrImplementationName,
			patterns: awsEcrPatterns,
		},
		{
			name:     AzureCrImplementationName,
			patterns: azureCrPatterns,
		},
		{
			name:     DockerHubImplementationName,
			patterns: dockerHubPatterns,
		},
		{
			name:     GcrImplementationName,
			patterns: gcrPatterns,
		},
		{
			name:     GitHubPackagesImplementationName,
			patterns: gitHubPackagesPatterns,
		},
		{
			name:     GitLabRegistryImplementationName,
			patterns: gitlabPatterns,
		},
		{
			name:     HarborImplementationName,
			patterns: harborPatterns,
		},
		{
			name:     QuayImplementationName,
			patterns: quayPatterns,
		},
	} {
		for _, pattern := range service.patterns {
			matched, err := regexp.MatchString(pattern, parsedResource.RegistryStr())
			if err != nil {
				return "", err
			}

			if matched {
				return service.name, nil
			}
		}
	}

	return "default", nil
}

func ImplementationList() []string {
	return []string{
		AwsEcrImplementationName,
		AzureCrImplementationName,
		DefaultImplementationName,
		DockerHubImplementationName,
		GcrImplementationName,
		GitHubPackagesImplementationName,
		GitLabRegistryImplementationName,
		HarborImplementationName,
		QuayImplementationName,
	}
}
