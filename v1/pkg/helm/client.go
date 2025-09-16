package helm

import (
	"context"
	"fmt"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/client-go/rest"

	"helm-charts-migrator/v1/pkg/logger"
)

type Client struct {
	actionConfig *action.Configuration
	settings     *cli.EnvSettings
	namespace    string
	log          *logger.NamedLogger
}

type ClientOptions struct {
	Namespace  string
	KubeConfig *rest.Config
	Context    string
}

func NewClient(opts ClientOptions) (*Client, error) {
	log := logger.WithName("helm-client")

	settings := cli.New()
	if opts.Context != "" {
		settings.KubeContext = opts.Context
	}

	actionConfig := new(action.Configuration)

	namespace := opts.Namespace
	if namespace == "" {
		namespace = "default"
	}

	err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), log.Info)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize helm action config: %w", err)
	}

	return &Client{
		actionConfig: actionConfig,
		settings:     settings,
		namespace:    namespace,
		log:          log,
	}, nil
}

func (c *Client) ListReleases() ([]*release.Release, error) {
	client := action.NewList(c.actionConfig)
	client.All = true
	client.AllNamespaces = false
	client.StateMask = action.ListAll // Include all states

	releases, err := client.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list helm releases: %w", err)
	}

	c.log.InfoS("Listed helm releases", "count", len(releases), "namespace", c.namespace)
	return releases, nil
}

func (c *Client) GetRelease(name string) (*release.Release, error) {
	client := action.NewGet(c.actionConfig)

	rel, err := client.Run(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get release %s: %w", name, err)
	}

	return rel, nil
}

func (c *Client) LoadChart(chartPath string) (*chart.Chart, error) {
	c.log.InfoS("Loading chart", "path", chartPath)

	chartObj, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	c.log.InfoS("Chart loaded successfully",
		"name", chartObj.Name(),
		"version", chartObj.Metadata.Version,
		"appVersion", chartObj.Metadata.AppVersion)

	return chartObj, nil
}

func (c *Client) InstallChart(_ context.Context, releaseName string, chartPath string, values map[string]interface{}) (*release.Release, error) {
	c.log.InfoS("Installing chart", "release", releaseName, "chart", chartPath, "namespace", c.namespace)

	chartObj, err := c.LoadChart(chartPath)
	if err != nil {
		return nil, err
	}

	client := action.NewInstall(c.actionConfig)
	client.ReleaseName = releaseName
	client.Namespace = c.namespace
	client.CreateNamespace = true

	rel, err := client.Run(chartObj, values)
	if err != nil {
		return nil, fmt.Errorf("failed to install chart: %w", err)
	}

	c.log.InfoS("Chart installed successfully",
		"release", rel.Name,
		"version", rel.Version,
		"status", rel.Info.Status)

	return rel, nil
}

func (c *Client) UpgradeChart(_ context.Context, releaseName string, chartPath string, values map[string]interface{}) (*release.Release, error) {
	c.log.InfoS("Upgrading chart", "release", releaseName, "chart", chartPath, "namespace", c.namespace)

	chartObj, err := c.LoadChart(chartPath)
	if err != nil {
		return nil, err
	}

	client := action.NewUpgrade(c.actionConfig)
	client.Namespace = c.namespace

	rel, err := client.Run(releaseName, chartObj, values)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade chart: %w", err)
	}

	c.log.InfoS("Chart upgraded successfully",
		"release", rel.Name,
		"version", rel.Version,
		"status", rel.Info.Status)

	return rel, nil
}

func (c *Client) UninstallRelease(releaseName string) error {
	c.log.InfoS("Uninstalling release", "release", releaseName, "namespace", c.namespace)

	client := action.NewUninstall(c.actionConfig)

	_, err := client.Run(releaseName)
	if err != nil {
		return fmt.Errorf("failed to uninstall release %s: %w", releaseName, err)
	}

	c.log.InfoS("Release uninstalled successfully", "release", releaseName)
	return nil
}
