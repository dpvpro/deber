package client

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types/swarm"
	"golang.org/x/net/context"
)

// SecretInspectWithRaw returns the secret information with raw data
func (cli *Client) SecretInspectWithRaw(ctx context.Context, id string) (swarm.Secret, []byte, error) {
	if err := cli.NewVersionError(ctx, "1.25", "secret inspect"); err != nil {
		return swarm.Secret{}, nil, err
	}
	resp, err := cli.get(ctx, "/secrets/"+id, nil, nil)
	defer ensureReaderClosed(resp)
	if err != nil {
		return swarm.Secret{}, nil, err
	}

	body, err := io.ReadAll(resp.body)
	if err != nil {
		return swarm.Secret{}, nil, err
	}

	var secret swarm.Secret
	rdr := bytes.NewReader(body)
	err = json.NewDecoder(rdr).Decode(&secret)

	return secret, body, err
}
