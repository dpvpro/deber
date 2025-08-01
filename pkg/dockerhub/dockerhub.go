// Package dockerhub includes DockerHub API wrappers
package dockerhub

import (
	// "encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/thedevsaddam/gojsonq"
)

// GetTags function queries DockerHub API for a list of all
// available tags of a given repository.
//
// https://stackoverflow.com/questions/48856693/dockerhub-api-listing-tags
// curl -s GET 'https://hub.docker.com/v2/repositories/library/debian/tags?page_size=1000' | jq -r '.results|.[]|.name
func GetTags(repo string) ([]string, error) {

	var tags []string

	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/library/%s/tags?page_size=1000", repo)

	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	err = response.Body.Close()
	if err != nil {
		return nil, err
	}

	jsonRaw := string(bytes)

	jq := gojsonq.New().FromString(jsonRaw)
	if jq.Error() != nil {
		return nil, err
	}

	res, err := jq.From("results").PluckR("name")
	if err != nil {
		return nil, err
	}

	tags, _ = res.StringSlice()

	return tags, nil
}

// MatchRepo returns repo which has the given tag
func MatchRepo(repos []string, tag string) (string, error) {
	for _, repo := range repos {
		tags, err := GetTags(repo)
		if err != nil {
			return "", err
		}

		if slices.Contains(tags, tag) {
			return repo, nil
		}
	}

	return "", errors.New("couldn't match tag with repo")

}
