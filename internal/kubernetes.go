package internal

import (
	"log"
	"os"
	"path/filepath"

	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	"github.com/sirupsen/logrus"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ContextType string

var (
	KubeCliCtxType    ContextType = "kubeCli"
	PodmanCliCtxType  ContextType = "podmanCli"
	KubeConfigCtxType ContextType = "kubeconfig"
	NamespaceCtxType  ContextType = "namespace"
)

func GetDefaultKubeconfigPath() string {
	if k8sconfig, ok := os.LookupEnv("KUBECONFIG"); ok {
		return k8sconfig
	}
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return "$HOME/.kube/config"

}

func GetCurrentNamespace() string {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = GetDefaultKubeconfigPath()
	// Load the config
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, configOverrides)

	// Retrieve the namespace
	namespace, _, err := kubeConfig.Namespace()
	if err != nil {
		logrus.Warnf("failed to retrieve current namespace from the kubeconfig context. Defaulting to 'default' :%s", err)
		return "default"
	}

	return namespace
}

func GetClient(kubeConfigPath string) (client.Client, error) {

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic(err.Error())
	}
	client, err := client.New(config, client.Options{})
	// create the clientset
	if err != nil {
		return nil, err
	}
	err = applicationapiv1alpha1.AddToScheme(client.Scheme())
	if err != nil {
		logrus.Fatalf("unable to add applicationapiv1alpha1 schema to client: %s", err)
		return nil, err
	}
	err = releaseapiv1alpha1.AddToScheme(client.Scheme())
	if err != nil {
		logrus.Fatalf("unable to add releaseapiv1alpha1 schema to client: %s", err)
		return nil, err
	}

	return client, nil
}

func GetDynamicClient(kubeConfigPath string) (*dynamic.DynamicClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic(err.Error())
	}
	// Create the dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}
	return dynamicClient, nil
}
