package integrationtests

// This file includes utilities to start an environment with API Server, and a configured reconciler.

import (
	"context"
	"fmt"
	"go/build"
	"os"
	"path/filepath"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	DefaultTestAPIURL    = "https://ENVIRONMENTID.live.dynatrace.com/api"
	DefaultTestNamespace = "dynatrace"
)

func init() {
	os.Setenv("POD_NAMESPACE", DefaultTestNamespace)
}

type ControllerTestEnvironment struct {
	CommunicationHosts []string
	Client             client.Client
	Reconciler         *dynakube.DynakubeController

	server *envtest.Environment
}

func newTestEnvironment() (*ControllerTestEnvironment, error) {
	environment := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "config", "crd", "bases"),
			// ToDo: currently this is the only way to get the CRD - see https://github.com/kubernetes-sigs/controller-runtime/pull/1393
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "istio.io", "api@v0.0.0-20211020081732-2de5b65af1fe", "kubernetes"),
		},
	}
	kubernetesAPIServer := environment.ControlPlane.GetAPIServer()

	arguments := kubernetesAPIServer.Configure()
	arguments.Set("--allow-privileged")

	cfg, err := environment.Start()
	if err != nil {
		return nil, err
	}

	kubernetesClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		errStop := kubernetesAPIServer.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%s\n%s", err.Error(), errStop.Error())
		}
		return nil, err
	}

	if err = kubernetesClient.Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultTestNamespace,
		}}); err != nil {
		errStop := kubernetesAPIServer.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%s\n%s", err.Error(), errStop.Error())
		}
		return nil, err
	}

	if err = kubernetesClient.Create(context.TODO(), buildDynatraceClientSecret()); err != nil {
		errStop := kubernetesAPIServer.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%s\n%s", err.Error(), errStop.Error())
		}
		return nil, err
	}

	communicationHosts := []string{
		"https://endpoint1.test.com/communication",
		"https://endpoint2.test.com/communication",
	}
	testEnvironment := &ControllerTestEnvironment{
		server:             environment,
		Client:             kubernetesClient,
		CommunicationHosts: communicationHosts,
	}
	testEnvironment.Reconciler = dynakube.NewDynaKubeController(
		kubernetesClient, kubernetesClient, scheme.Scheme,
		mockDynatraceClientFunc(&testEnvironment.CommunicationHosts), cfg)

	return testEnvironment, nil
}

func (e *ControllerTestEnvironment) Stop() error {
	return e.server.Stop()
}

func (e *ControllerTestEnvironment) AddOneAgent(n string, s *dynatracev1beta1.DynaKubeSpec) error {
	return e.Client.Create(context.TODO(), &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n,
			Namespace: DefaultTestNamespace,
		},
		Spec: *s,
	})
}

func newReconciliationRequest(oaName string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      oaName,
			Namespace: DefaultTestNamespace,
		},
	}
}

func mockDynatraceClientFunc(communicationHosts *[]string) dynakube.DynatraceClientFunc {
	return func(dynakube.DynatraceClientProperties) (dtclient.Client, error) {
		commHosts := make([]dtclient.CommunicationHost, len(*communicationHosts))
		for i, c := range *communicationHosts {
			commHosts[i] = dtclient.CommunicationHost{Protocol: "https", Host: c, Port: 443}
		}

		connInfo := dtclient.ConnectionInfo{
			TenantUUID:         "asdf",
			CommunicationHosts: commHosts,
		}

		dtc := new(dtclient.MockDynatraceClient)
		dtc.On("GetLatestAgentVersion", "unix", "default").Return("17", nil)
		dtc.On("GetLatestAgentVersion", "unix", "paas").Return("18", nil)
		dtc.On("GetConnectionInfo").Return(connInfo, nil)
		dtc.On("GetCommunicationHostForClient").Return(dtclient.CommunicationHost{
			Protocol: "https",
			Host:     DefaultTestAPIURL,
			Port:     443,
		}, nil)
		dtc.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtc.On("GetTokenScopes", "43").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		return dtc, nil
	}
}

func buildDynatraceClientSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token-test",
			Namespace: DefaultTestNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"paasToken": []byte("42"),
			"apiToken":  []byte("43"),
		},
	}
}
