package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn/kubernetes"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	ootov1alpha1 "github.com/qbarrand/oot-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -source=getter.go -package=registry -destination=mock_getter.go

type Getter interface {
	ImageExists(ctx context.Context, containerImage string, po ootov1alpha1.PullOptions, ns string) (bool, error)
}

type getter struct {
	kubeClient client.Client
}

func NewGetter(kubeClient client.Client) Getter {
	return &getter{kubeClient: kubeClient}
}

func (g *getter) ImageExists(
	ctx context.Context,
	containerImage string,
	po ootov1alpha1.PullOptions,
	namespace string,
) (bool, error) {
	ref, err := name.ParseReference(containerImage)
	if err != nil {
		return false, fmt.Errorf("could not parse the container image name: %v", err)
	}

	opts := []remote.Option{
		remote.WithContext(ctx),
	}

	if po.Insecure {
		rt := http.DefaultTransport.(*http.Transport).Clone()
		rt.TLSClientConfig.InsecureSkipVerify = true

		opts = append(
			opts,
			remote.WithTransport(rt),
		)
	}

	if secretName := po.Secret.Name; secretName != "" {
		secret := v1.Secret{}
		key := types.NamespacedName{Namespace: namespace, Name: secretName}

		if err := g.kubeClient.Get(ctx, key, &secret); err != nil {
			return false, fmt.Errorf("could not fetch pull secret %q: %v", key, err)
		}

		keychain, err := kubernetes.NewFromPullSecrets(ctx, []v1.Secret{secret})
		if err != nil {
			return false, fmt.Errorf("could not create a keychain from %q: %v", key, err)
		}

		opts = append(
			opts,
			remote.WithAuthFromKeychain(keychain),
		)
	}

	if _, err = remote.Get(ref, opts...); err != nil {
		te := &transport.Error{}

		if errors.As(err, &te) && te.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("could not get image: %v", err)
	}

	return true, nil
}
