package client

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroClient "github.com/vmware-tanzu/velero/pkg/client"
	clientset "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type env int

const (
	product env = iota
	disaster
)

type Factory interface {
	// BindFlags binds common flags (--kubeconfig, --namespace) to the passed-in FlagSet.
	BindFlags(flags *pflag.FlagSet)
	// Client returns a VeleroClient.
	Client() (clientset.Interface, error)
	// KubeClient returns a Kubernetes client which to be health checked.
	KubeClient() (kubernetes.Interface, error)
	// SetClientQPS sets the Queries Per Second for a client.
	SetClientQPS(float32)
	// SetClientBurst sets the Burst for a client.
	SetClientBurst(int)
	// Namespace return velero running namespace
	Namespace() string
}

type factory struct {
	flags               *pflag.FlagSet
	productKubeConfig   string
	productKubeContext  string
	namespace           string
	disasterKubeConfig  string
	disasterKubeContext string
	baseName            string
	clientQPS           float32
	clientBurst         int
}

func NewFactory(name string) Factory {
	f := &factory{
		flags:    pflag.NewFlagSet("", pflag.ContinueOnError),
		baseName: name,
	}

	f.namespace = os.Getenv("VELERO_NAMESPACE")

	// We didn't get the namespace via env var, so use the default.
	// Command line flags will override when BindFlags is called.
	if f.namespace == "" {
		f.namespace = velerov1api.DefaultNamespace
	}

	f.flags.StringVar(&f.productKubeConfig, "product-kubeconfig", "", "Path to the kubeconfig file to use to talk to the product Kubernetes apiserver. If unset, try the in-cluster configuration")
	f.flags.StringVar(&f.productKubeContext, "product-kubecontext", "", "The context to use to talk to the product Kubernetes apiserver. If unset defaults to whatever your current-context is (kubectl config current-context)")
	f.flags.StringVar(&f.namespace, "namespace", f.namespace, "The namespace in which Takeover should operate")
	f.flags.StringVar(&f.disasterKubeConfig, "disaster-kubeconfig", "", "Path to the kubeconfig file to use to talk to the disaster Kubernetes apiserver. If unset, try the in-cluster configuration")
	f.flags.StringVar(&f.disasterKubeContext, "disaster-kubecontext", "", "The context to use to talk to the disaster Kubernetes apiserver. If unset defaults to whatever your current-context is (kubectl config current-context)")

	return f
}

func (f *factory) BindFlags(flags *pflag.FlagSet) {
	flags.AddFlagSet(f.flags)
}

func (f *factory) ClientConfig(e env) (*rest.Config, error) {
	if e == disaster {
		return veleroClient.Config(f.disasterKubeConfig, f.disasterKubeContext, f.baseName, f.clientQPS, f.clientBurst)
	}
	return veleroClient.Config(f.productKubeConfig, f.productKubeContext, f.baseName, f.clientQPS, f.clientBurst)
}

func (f *factory) Client() (clientset.Interface, error) {
	clientConfig, err := f.ClientConfig(disaster)
	if err != nil {
		return nil, err
	}

	veleroClient, err := clientset.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return veleroClient, nil
}

func (f *factory) KubeClient() (kubernetes.Interface, error) {
	clientConfig, err := f.ClientConfig(product)
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return kubeClient, nil
}

func (f *factory) SetClientQPS(qps float32) {
	f.clientQPS = qps
}

func (f *factory) SetClientBurst(burst int) {
	f.clientBurst = burst
}

func (f *factory) Namespace() string {
	return f.namespace
}
