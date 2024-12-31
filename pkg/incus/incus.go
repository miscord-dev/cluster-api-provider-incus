package incus

import (
	"context"
	"fmt"
	"maps"
	"net/http"

	incusclient "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
	infrav1alpha1 "github.com/miscord-dev/cluster-api-provider-incus/api/v1alpha1"
)

var (
	ErrorInstanceNotFound = fmt.Errorf("instance not found")
)

type BootstrapData struct {
	Format string
	Data   string
}

type CreateInstanceInput struct {
	Name          string
	BootstrapData BootstrapData

	infrav1alpha1.InstanceSpec
}

type GetInstanceOutput struct {
	Name string
	infrav1alpha1.InstanceSpec

	// TODO: Add status
	StatusCode api.StatusCode
}

type Client interface {
	CreateInstance(ctx context.Context, spec CreateInstanceInput) error
	InstanceExists(ctx context.Context, name string) (bool, error)
	GetInstance(ctx context.Context, name string) (*GetInstanceOutput, error)
	DeleteInstance(ctx context.Context, name string) error
	StopInstance(ctx context.Context, name string) error
}

type client struct {
	client incusclient.InstanceServer
}

func NewClient(instanceServer incusclient.InstanceServer) Client {
	return &client{
		client: instanceServer,
	}
}

func (c *client) CreateInstance(ctx context.Context, spec CreateInstanceInput) error {
	config := maps.Clone(spec.Config)

	switch spec.BootstrapData.Format {
	case "cloud-config":
		config["cloud-init.user-data"] = spec.BootstrapData.Data
	case "":
		// Do nothing
	default:
		return fmt.Errorf("unsupported bootstrap data format: %s", spec.BootstrapData.Format)
	}

	req := api.InstancesPost{
		Name: spec.Name,
		InstancePut: api.InstancePut{
			Architecture: spec.Architecture,
			Config:       spec.Config,
			Devices:      spec.Devices,
			Ephemeral:    spec.Ephemeral,
			Profiles:     spec.Profiles,
			Restore:      spec.Restore,
			Stateful:     spec.Stateful,
			Description:  spec.Description,
		},
		Source: api.InstanceSource{
			Type:              spec.Source.Type,
			Certificate:       spec.Source.Certificate,
			Alias:             spec.Source.Alias,
			Fingerprint:       spec.Source.Fingerprint,
			Properties:        spec.Source.Properties,
			Server:            spec.Source.Server,
			Secret:            spec.Source.Secret,
			Protocol:          spec.Source.Protocol,
			BaseImage:         spec.Source.BaseImage,
			Mode:              spec.Source.Mode,
			Operation:         spec.Source.Operation,
			Websockets:        spec.Source.Websockets,
			Source:            spec.Source.Source,
			Live:              spec.Source.Live,
			InstanceOnly:      spec.Source.InstanceOnly,
			Refresh:           spec.Source.Refresh,
			Project:           spec.Source.Project,
			AllowInconsistent: spec.Source.AllowInconsistent,
		},
		Type:  api.InstanceType(spec.Type),
		Start: true,
	}

	_, err := c.client.CreateInstance(req)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	return nil
}

func (c *client) InstanceExists(ctx context.Context, name string) (bool, error) {
	_, _, err := c.client.GetInstance(name)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get instance: %w", err)
	}

	return true, nil
}

func (c *client) GetInstance(ctx context.Context, name string) (*GetInstanceOutput, error) {
	resp, _, err := c.client.GetInstanceFull(name)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			return nil, ErrorInstanceNotFound
		}

		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return &GetInstanceOutput{
		Name: name,
		InstanceSpec: infrav1alpha1.InstanceSpec{
			Architecture: resp.Architecture,
			Config:       resp.Config,
			Devices:      resp.Devices,
			Ephemeral:    resp.Ephemeral,
			Profiles:     resp.Profiles,
			Restore:      resp.Restore,
			Stateful:     resp.Stateful,
			Description:  resp.Description,
			Type:         infrav1alpha1.InstanceType(resp.Type),
		},
		StatusCode: resp.StatusCode,
	}, nil
}

func (c *client) DeleteInstance(ctx context.Context, name string) error {
	_, err := c.client.DeleteInstance(name)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}

func (c *client) StopInstance(ctx context.Context, name string) error {
	_, err := c.client.UpdateInstanceState(name, api.InstanceStatePut{
		Action: "stop",
	}, "")
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return nil
}
