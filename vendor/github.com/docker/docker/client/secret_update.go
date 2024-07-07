package client

import (
	"net/url"

	"github.com/docker/docker/api/types/swarm"
	"golang.org/x/net/context"
)

// SecretUpdate attempts to update a secret.
func (cli *Client) SecretUpdate(ctx context.Context, id string, version swarm.Version, secret swarm.SecretSpec) error {
	if err := cli.NewVersionError(ctx, "1.25", "secret update"); err != nil {
		return err
	}
	query := url.Values{}
	query.Set("version", version.String())
	resp, err := cli.post(ctx, "/secrets/"+id+"/update", query, secret, nil)
	ensureReaderClosed(resp)
	return err
}
