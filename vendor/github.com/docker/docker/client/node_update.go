package client

import (
	"net/url"

	"github.com/docker/docker/api/types/swarm"
	"golang.org/x/net/context"
)

// NodeUpdate updates a Node.
func (cli *Client) NodeUpdate(ctx context.Context, nodeID string, version swarm.Version, node swarm.NodeSpec) error {
	query := url.Values{}
	query.Set("version", version.String())
	resp, err := cli.post(ctx, "/nodes/"+nodeID+"/update", query, node, nil)
	ensureReaderClosed(resp)
	return err
}
