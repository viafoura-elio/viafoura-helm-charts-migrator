package services

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"helm-charts-migrator/v1/pkg/logger"
)

// kubernetesService implements KubernetesService interface
type kubernetesService struct {
	log *logger.NamedLogger
}

// NewKubernetesService creates a new KubernetesService
func NewKubernetesService() KubernetesService {
	return &kubernetesService{
		log: logger.WithName("kubernetes-service"),
	}
}

// GetClient returns a Kubernetes client for the given context
func (k *kubernetesService) GetClient(kubeContext string) (*kubernetes.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{
		CurrentContext: kubeContext,
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// ListReleases lists all Helm releases in a namespace
func (k *kubernetesService) ListReleases(ctx context.Context, kubeContext, namespace string) ([]*release.Release, error) {
	settings := cli.New()
	settings.KubeContext = kubeContext

	actionConfig := new(action.Configuration)
	logFunc := func(format string, v ...interface{}) {
		k.log.V(3).InfoS(fmt.Sprintf(format, v...), "component", "helm")
	}
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", logFunc); err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	listAction := action.NewList(actionConfig)
	listAction.All = true
	listAction.Deployed = true

	releases, err := listAction.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	k.log.V(2).InfoS("Listed releases", "context", kubeContext, "namespace", namespace, "count", len(releases))
	return releases, nil
}

// GetRelease gets a specific Helm release
func (k *kubernetesService) GetRelease(ctx context.Context, kubeContext, namespace, releaseName string) (*release.Release, error) {
	settings := cli.New()
	settings.KubeContext = kubeContext

	actionConfig := new(action.Configuration)
	logFunc := func(format string, v ...interface{}) {
		k.log.V(3).InfoS(fmt.Sprintf(format, v...), "component", "helm")
	}
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", logFunc); err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	getAction := action.NewGet(actionConfig)
	helmRelease, err := getAction.Run(releaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get helmRelease %s: %w", releaseName, err)
	}

	return helmRelease, nil
}

// SwitchContext switches the kubectl context
func (k *kubernetesService) SwitchContext(context string) error {
	cmd := exec.Command("kubectl", "config", "use-context", context)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to switch context to %s: %w\nOutput: %s", context, err, output)
	}

	k.log.V(2).InfoS("Switched kubectl context", "context", context)
	return nil
}

// GetCurrentContext returns the current kubectl context
func (k *kubernetesService) GetCurrentContext() (string, error) {
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current context: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
