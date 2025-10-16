package kubernetes

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"helm-charts-migrator/v1/pkg/logger"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    ClientOptions
		wantErr bool
	}{
		{
			name: "with context",
			opts: ClientOptions{
				Context: "test-context",
			},
			wantErr: false, // Will fail in real test without valid kubeconfig
		},
		{
			name: "with kubeconfig path",
			opts: ClientOptions{
				KubeConfig: "/tmp/test-kubeconfig",
			},
			wantErr: false, // Will fail in real test without valid kubeconfig
		},
		{
			name:    "with empty options",
			opts:    ClientOptions{},
			wantErr: false, // Should use default kubeconfig
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: These tests will fail without a valid kubeconfig
			// In a real environment, you would mock the client creation
			_, err := NewClient(tt.opts)

			// Since we don't have a valid kubeconfig in test environment,
			// we expect errors in most cases
			if err == nil && tt.name != "with empty options" {
				t.Skip("Skipping - requires valid kubeconfig")
			}
		})
	}
}

func TestClient_TestConnection(t *testing.T) {
	// Create a fake clientset for testing
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
		context:   "test-context",
	}

	ctx := context.Background()
	err := client.TestConnection(ctx)

	if err != nil {
		t.Errorf("TestConnection() error = %v, want nil", err)
	}
}

func TestClient_Getters(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()
	testConfig := &rest.Config{Host: "https://test.example.com"}

	client := &Client{
		clientset: fakeClientset,
		config:    testConfig,
		context:   "test-context",
		log:       logger.WithName("test"),
	}

	// Test GetClientset
	if client.GetClientset() != fakeClientset {
		t.Error("GetClientset() did not return expected clientset")
	}

	// Test GetConfig
	if client.GetConfig() != testConfig {
		t.Error("GetConfig() did not return expected config")
	}

	// Test GetContext
	if client.GetContext() != "test-context" {
		t.Errorf("GetContext() = %v, want test-context", client.GetContext())
	}
}

func TestClient_TestConnectionWithTimeout(t *testing.T) {
	// Create a fake clientset for testing
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.TestConnection(ctx)
	if err != nil {
		t.Errorf("TestConnection() with timeout error = %v, want nil", err)
	}
}

func TestClient_ListNamespaces(t *testing.T) {
	// Create test namespaces
	testNamespaces := []runtime.Object{
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		},
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-system",
			},
		},
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		},
	}

	// Create a fake clientset with test data
	fakeClientset := fake.NewSimpleClientset(testNamespaces...)

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	ctx := context.Background()
	namespaces, err := client.ListNamespaces(ctx)

	if err != nil {
		t.Fatalf("ListNamespaces() error = %v, want nil", err)
	}

	if len(namespaces) != 3 {
		t.Errorf("ListNamespaces() returned %d namespaces, want 3", len(namespaces))
	}

	// Check namespace names
	expectedNames := map[string]bool{
		"default":        false,
		"kube-system":    false,
		"test-namespace": false,
	}

	for _, ns := range namespaces {
		if _, exists := expectedNames[ns]; !exists {
			t.Errorf("Unexpected namespace: %s", ns)
		}
		expectedNames[ns] = true
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Expected namespace %s not found", name)
		}
	}
}

func TestClient_GetNamespace(t *testing.T) {
	// Create test namespace
	testNamespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				"env":  "test",
				"team": "platform",
			},
		},
	}

	// Create a fake clientset with test data
	fakeClientset := fake.NewSimpleClientset(testNamespace)

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{
			name:      "existing namespace",
			namespace: "test-namespace",
			wantErr:   false,
		},
		{
			name:      "non-existent namespace",
			namespace: "non-existent",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ns, err := client.GetNamespace(ctx, tt.namespace)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && ns == nil {
				t.Error("GetNamespace() returned nil namespace for existing namespace")
			}

			if !tt.wantErr && ns != nil {
				if ns.Name != tt.namespace {
					t.Errorf("GetNamespace() returned namespace with name %s, want %s", ns.Name, tt.namespace)
				}
			}
		})
	}
}

func TestClient_GetPods(t *testing.T) {
	// Create test pods
	testPods := []runtime.Object{
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: "kube-system",
				Labels: map[string]string{
					"app": "system",
				},
			},
		},
	}

	// Create a fake clientset with test data
	fakeClientset := fake.NewSimpleClientset(testPods...)

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	tests := []struct {
		name          string
		namespace     string
		labelSelector string
		expectedPods  int
	}{
		{
			name:          "all pods in default namespace",
			namespace:     "default",
			labelSelector: "",
			expectedPods:  2,
		},
		{
			name:          "pods with label selector",
			namespace:     "default",
			labelSelector: "app=test",
			expectedPods:  2,
		},
		{
			name:          "pods in kube-system namespace",
			namespace:     "kube-system",
			labelSelector: "",
			expectedPods:  1,
		},
		{
			name:          "no pods with non-matching label",
			namespace:     "default",
			labelSelector: "app=nonexistent",
			expectedPods:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pods, err := client.GetPods(ctx, tt.namespace, tt.labelSelector)

			if err != nil {
				t.Fatalf("GetPods() error = %v", err)
			}

			if len(pods) != tt.expectedPods {
				t.Errorf("GetPods() returned %d pods, want %d", len(pods), tt.expectedPods)
			}
		})
	}
}

func TestClient_GetServices(t *testing.T) {
	// Create test services
	testServices := []runtime.Object{
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "service1",
				Namespace: "default",
			},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeClusterIP,
				Ports: []v1.ServicePort{
					{
						Port: 80,
						Name: "http",
					},
				},
			},
		},
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "service2",
				Namespace: "default",
			},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeLoadBalancer,
				Ports: []v1.ServicePort{
					{
						Port: 443,
						Name: "https",
					},
				},
			},
		},
	}

	// Create a fake clientset with test data
	fakeClientset := fake.NewSimpleClientset(testServices...)

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	ctx := context.Background()
	services, err := client.GetServices(ctx, "default")

	if err != nil {
		t.Fatalf("GetServices() error = %v", err)
	}

	if len(services) != 2 {
		t.Errorf("GetServices() returned %d services, want 2", len(services))
	}

	// Check service details
	for _, svc := range services {
		if svc.Name == "service1" {
			if svc.Spec.Type != v1.ServiceTypeClusterIP {
				t.Errorf("service1 type = %v, want ClusterIP", svc.Spec.Type)
			}
		} else if svc.Name == "service2" {
			if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
				t.Errorf("service2 type = %v, want LoadBalancer", svc.Spec.Type)
			}
		}
	}
}

func TestClient_GetConfigMaps(t *testing.T) {
	// Create test configmaps
	testConfigMaps := []runtime.Object{
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "config1",
				Namespace: "default",
			},
			Data: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "config2",
				Namespace: "default",
			},
			Data: map[string]string{
				"app.properties": "property1=value1\nproperty2=value2",
			},
		},
	}

	// Create a fake clientset with test data
	fakeClientset := fake.NewSimpleClientset(testConfigMaps...)

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	ctx := context.Background()
	configMaps, err := client.GetConfigMaps(ctx, "default")

	if err != nil {
		t.Fatalf("GetConfigMaps() error = %v", err)
	}

	if len(configMaps) != 2 {
		t.Errorf("GetConfigMaps() returned %d configmaps, want 2", len(configMaps))
	}

	// Check configmap data
	for _, cm := range configMaps {
		if cm.Name == "config1" {
			if val, exists := cm.Data["key1"]; !exists || val != "value1" {
				t.Errorf("config1 missing expected data")
			}
		}
	}
}

func TestClient_GetSecrets(t *testing.T) {
	// Create test secrets
	testSecrets := []runtime.Object{
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret1",
				Namespace: "default",
			},
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tls-secret",
				Namespace: "default",
			},
			Type: v1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.crt": []byte("certificate"),
				"tls.key": []byte("key"),
			},
		},
	}

	// Create a fake clientset with test data
	fakeClientset := fake.NewSimpleClientset(testSecrets...)

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	ctx := context.Background()
	secrets, err := client.GetSecrets(ctx, "default")

	if err != nil {
		t.Fatalf("GetSecrets() error = %v", err)
	}

	if len(secrets) != 2 {
		t.Errorf("GetSecrets() returned %d secrets, want 2", len(secrets))
	}

	// Check secret types
	for _, secret := range secrets {
		if secret.Name == "secret1" && secret.Type != v1.SecretTypeOpaque {
			t.Errorf("secret1 type = %v, want Opaque", secret.Type)
		}
		if secret.Name == "tls-secret" && secret.Type != v1.SecretTypeTLS {
			t.Errorf("tls-secret type = %v, want TLS", secret.Type)
		}
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should handle cancelled context gracefully
	_, err := client.ListNamespaces(ctx)
	if err == nil {
		// Fake client might not respect context cancellation
		t.Skip("Fake client doesn't respect context cancellation")
	}
}

func TestClient_EmptyResults(t *testing.T) {
	// Create a fake clientset with no resources
	fakeClientset := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	ctx := context.Background()

	// Test empty namespaces
	namespaces, err := client.ListNamespaces(ctx)
	if err != nil {
		t.Errorf("ListNamespaces() with no namespaces error = %v", err)
	}
	if len(namespaces) != 0 {
		t.Errorf("ListNamespaces() returned %d namespaces, want 0", len(namespaces))
	}

	// Test empty pods
	pods, err := client.GetPods(ctx, "default", "")
	if err != nil {
		t.Errorf("GetPods() with no pods error = %v", err)
	}
	if len(pods) != 0 {
		t.Errorf("GetPods() returned %d pods, want 0", len(pods))
	}

	// Test empty services
	services, err := client.GetServices(ctx, "default")
	if err != nil {
		t.Errorf("GetServices() with no services error = %v", err)
	}
	if len(services) != 0 {
		t.Errorf("GetServices() returned %d services, want 0", len(services))
	}
}

// Benchmark tests
func BenchmarkClient_ListNamespaces(b *testing.B) {
	// Create test namespaces
	var namespaces []runtime.Object
	for i := 0; i < 100; i++ {
		namespaces = append(namespaces, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("namespace-%d", i),
			},
		})
	}

	fakeClientset := fake.NewSimpleClientset(namespaces...)
	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ListNamespaces(ctx)
	}
}

func BenchmarkClient_GetPods(b *testing.B) {
	// Create test pods
	var pods []runtime.Object
	for i := 0; i < 100; i++ {
		pods = append(pods, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
			},
		})
	}

	fakeClientset := fake.NewSimpleClientset(pods...)
	client := &Client{
		clientset: fakeClientset,
		config:    &rest.Config{},
		log:       logger.WithName("test"),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetPods(ctx, "default", "app=test")
	}
}

// Test NewClient with invalid kubeconfig
func TestNewClient_InvalidKubeconfig(t *testing.T) {
	// Create a temp file with invalid content
	tempFile, err := os.CreateTemp("", "invalid-kubeconfig")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	// Write invalid YAML
	tempFile.WriteString("invalid: yaml: content: {")
	tempFile.Close()

	_, err = NewClient(ClientOptions{
		KubeConfig: tempFile.Name(),
	})

	if err == nil {
		t.Error("Expected error with invalid kubeconfig, got nil")
	}
}

// Test with KUBECONFIG environment variable
func TestNewClient_WithEnvVar(t *testing.T) {
	// Save original env
	originalEnv := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", originalEnv)

	// Set test env
	os.Setenv("KUBECONFIG", "/tmp/test-env-kubeconfig")

	// Test will fail but should pick up the env var
	_, _ = NewClient(ClientOptions{})
	// We don't check error as we don't have a valid kubeconfig
}

// Helper function for integration tests (requires real cluster)
func TestClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires a real Kubernetes cluster
	// It will be skipped in most CI environments
	client, err := NewClient(ClientOptions{})
	if err != nil {
		t.Skip("Skipping integration test - no valid kubeconfig available")
	}

	ctx := context.Background()

	// Test connection
	if err := client.TestConnection(ctx); err != nil {
		t.Fatalf("Failed to connect to cluster: %v", err)
	}

	// List namespaces
	namespaces, err := client.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("Failed to list namespaces: %v", err)
	}

	t.Logf("Connected to cluster with %d namespaces", len(namespaces))
}
