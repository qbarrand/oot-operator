package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	ootov1alpha1 "github.com/qbarrand/oot-operator/api/v1alpha1"
)

//go:generate mockgen -source=getter.go -package=registry -destination=mock_getter.go

type Getter interface {
	ImageExists(ctx context.Context, containerImage string, po ootov1alpha1.PullOptions) (bool, error)
}

type getter struct{}

func NewGetter() Getter {
	return &getter{}
}

func (getter) ImageExists(ctx context.Context, containerImage string, po ootov1alpha1.PullOptions) (bool, error) {
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

		remote.WithTransport(rt)
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
