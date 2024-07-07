package client

import (
	"io"
	"net/http"
	"net/url"

	"github.com/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// PluginUpgrade upgrades a plugin
func (cli *Client) PluginUpgrade(ctx context.Context, name string, options types.PluginInstallOptions) (rc io.ReadCloser, err error) {
	if err := cli.NewVersionError(ctx, "1.26", "plugin upgrade"); err != nil {
		return nil, err
	}
	query := url.Values{}
	if _, err := reference.ParseNormalizedNamed(options.RemoteRef); err != nil {
		return nil, errors.Wrap(err, "invalid remote reference")
	}
	query.Set("remote", options.RemoteRef)

	privileges, err := cli.checkPluginPermissions(ctx, query, options)
	if err != nil {
		return nil, err
	}

	resp, err := cli.tryPluginUpgrade(ctx, query, privileges, name, options.RegistryAuth)
	if err != nil {
		return nil, err
	}
	return resp.body, nil
}

func (cli *Client) tryPluginUpgrade(ctx context.Context, query url.Values, privileges types.PluginPrivileges, name, registryAuth string) (serverResponse, error) {
	return cli.post(ctx, "/plugins/"+name+"/upgrade", query, privileges, http.Header{
		registry.AuthHeader: {registryAuth},
	})
}
