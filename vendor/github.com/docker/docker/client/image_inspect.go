package client

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
)

// ImageInspectWithRaw returns the image information and its raw representation.
func (cli *Client) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	serverResp, err := cli.get(ctx, "/images/"+imageID+"/json", nil, nil)
	defer ensureReaderClosed(serverResp)
	if err != nil {
		return types.ImageInspect{}, nil, err
	}

	body, err := io.ReadAll(serverResp.body)
	if err != nil {
		return types.ImageInspect{}, nil, err
	}

	var response types.ImageInspect
	rdr := bytes.NewReader(body)
	err = json.NewDecoder(rdr).Decode(&response)
	return response, body, err
}
