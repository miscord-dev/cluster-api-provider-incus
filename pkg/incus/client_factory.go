package incus

import (
	"context"
	"fmt"

	incusclient "github.com/lxc/incus/client"
	infrav1alpha1 "github.com/miscord-dev/cluster-api-provider-incus/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientFactory interface {
	GetClientForCluster(ctx context.Context, cluster *infrav1alpha1.IncusCluster) (Client, error)
}

type clientFactory struct {
	k8sClient ctrlclient.Client
}

func NewClientFactory(k8sClient ctrlclient.Client) ClientFactory {
	return &clientFactory{
		k8sClient: k8sClient,
	}
}

func (f *clientFactory) GetClientForCluster(ctx context.Context, cluster *infrav1alpha1.IncusCluster) (Client, error) {
	connectionArgs, err := f.buildConnectionArgs(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to build connection args: %w", err)
	}

	incusServer, err := incusclient.ConnectIncus(cluster.Spec.Incus.Endpoint, connectionArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Incus server %s: %w", cluster.Spec.Incus.Endpoint, err)
	}

	if cluster.Spec.Incus.Project != "" {
		incusServer = incusServer.UseProject(cluster.Spec.Incus.Project)
	}

	return NewClient(incusServer), nil
}

func (f *clientFactory) buildConnectionArgs(ctx context.Context, cluster *infrav1alpha1.IncusCluster) (*incusclient.ConnectionArgs, error) {
	secret := &corev1.Secret{}
	secretKey := ctrlclient.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Spec.Incus.SecretRef.Name,
	}

	if err := f.k8sClient.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", cluster.Spec.Incus.SecretRef.Name, err)
	}

	args := &incusclient.ConnectionArgs{}

	if tlsCA, ok := secret.Data["tls-ca"]; ok {
		args.TLSCA = string(tlsCA)
	}

	if tlsClientCert, ok := secret.Data["tls-client-cert"]; ok {
		args.TLSClientCert = string(tlsClientCert)
	}

	if tlsClientKey, ok := secret.Data["tls-client-key"]; ok {
		args.TLSClientKey = string(tlsClientKey)
	}

	if insecureSkipVerify, ok := secret.Data["insecure-skip-verify"]; ok {
		args.InsecureSkipVerify = string(insecureSkipVerify) == "true"
	}

	return args, nil
}