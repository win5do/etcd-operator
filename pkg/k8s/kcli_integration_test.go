package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/win5do/etcd-operator/pkg/test"
)

func TestMain(m *testing.M) {
	if test.ShouldRun() {
		m.Run()
	}
}

func TestFind(t *testing.T) {
	kcli := Kcli{
		client: test.Kcli(corev1.SchemeBuilder),
	}

	found := &corev1.ServiceAccount{}
	err := kcli.Find("foo", "default", found)
	require.NoError(t, err)
}

func TestDeleteALL(t *testing.T) {
	kcli := Kcli{
		client: test.Kcli(corev1.SchemeBuilder),
	}

	err := kcli.DeleteALLByLabel(&corev1.Service{}, "default", map[string]string{
		"svc": "export",
	})
	require.NoError(t, err)
}

func TestDeleteALLByLabel(t *testing.T) {
	kcli := Kcli{
		client: test.Kcli(corev1.SchemeBuilder),
	}

	err := kcli.DeleteALLByLabel(
		&corev1.PersistentVolumeClaim{},
		"default",
		map[string]string{
			"cr-name": "unit-test",
		},
	)
	require.NoError(t, err)
}
