package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"helm-charts-migrator/v1/pkg/logger"
)

type Client struct {
	clientset  kubernetes.Interface
	config     *rest.Config
	context    string
	kubeconfig string
	log        *logger.NamedLogger
}

type ClientOptions struct {
	KubeConfig string
	Context    string
}

func NewClient(opts ClientOptions) (*Client, error) {
	log := logger.WithName("k8s-client")

	kubeconfig := opts.KubeConfig
	if kubeconfig == "" {
		if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
			kubeconfig = envConfig
		} else if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	log.InfoS("Loading kubeconfig", "path", kubeconfig, "context", opts.Context)

	configLoadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	configOverrides := &clientcmd.ConfigOverrides{}

	if opts.Context != "" {
		configOverrides.CurrentContext = opts.Context
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		configOverrides,
	)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load raw config: %w", err)
	}

	currentContext := rawConfig.CurrentContext
	if opts.Context != "" {
		currentContext = opts.Context
	}

	return &Client{
		clientset:  clientset,
		config:     config,
		context:    currentContext,
		kubeconfig: kubeconfig,
		log:        log,
	}, nil
}

func (c *Client) GetClientset() kubernetes.Interface {
	return c.clientset
}

func (c *Client) GetConfig() *rest.Config {
	return c.config
}

func (c *Client) GetContext() string {
	return c.context
}

func (c *Client) TestConnection(ctx context.Context) error {
	c.log.Info("Testing kubernetes connection")

	version, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to kubernetes cluster: %w", err)
	}

	c.log.InfoS("Successfully connected to kubernetes cluster",
		"version", version.GitVersion,
		"platform", version.Platform,
		"context", c.context)

	return nil
}

func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	namespaceList, err := c.clientset.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	namespaces := make([]string, 0, len(namespaceList.Items))
	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, ns.Name)
	}

	return namespaces, nil
}

func (c *Client) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	namespace, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %w", name, err)
	}
	return namespace, nil
}

func (c *Client) GetPods(ctx context.Context, namespace string, labelSelector string) ([]*corev1.Pod, error) {
	listOptions := v1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}

	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	pods := make([]*corev1.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}

	return pods, nil
}

func (c *Client) GetServices(ctx context.Context, namespace string) ([]*corev1.Service, error) {
	serviceList, err := c.clientset.CoreV1().Services(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	services := make([]*corev1.Service, 0, len(serviceList.Items))
	for i := range serviceList.Items {
		services = append(services, &serviceList.Items[i])
	}

	return services, nil
}

func (c *Client) GetConfigMaps(ctx context.Context, namespace string) ([]*corev1.ConfigMap, error) {
	configMapList, err := c.clientset.CoreV1().ConfigMaps(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}

	configMaps := make([]*corev1.ConfigMap, 0, len(configMapList.Items))
	for i := range configMapList.Items {
		configMaps = append(configMaps, &configMapList.Items[i])
	}

	return configMaps, nil
}

func (c *Client) GetSecrets(ctx context.Context, namespace string) ([]*corev1.Secret, error) {
	secretList, err := c.clientset.CoreV1().Secrets(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	secrets := make([]*corev1.Secret, 0, len(secretList.Items))
	for i := range secretList.Items {
		secrets = append(secrets, &secretList.Items[i])
	}

	return secrets, nil
}
