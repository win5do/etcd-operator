package v1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuantityToInt(t *testing.T) {
	assert.Equal(t, int64(0), memQuantityToMiInt(""))
	assert.Equal(t, float64(-0.001), cpuQuantityToInt(""))

	assert.Equal(t, int64(1024), memQuantityToMiInt("1024Mi"))
	assert.Equal(t, int64(8192), memQuantityToMiInt("8Gi"))
	assert.Equal(t, int64(256), memQuantityToMiInt(fmt.Sprintf("%dKi", 256*1024)))

	assert.Equal(t, float64(4), cpuQuantityToInt("4"))
	assert.Equal(t, float64(4), cpuQuantityToInt("4000m"))
	assert.Equal(t, float64(0.4), cpuQuantityToInt("400m"))
}

func TestDefault(t *testing.T) {
	in := &Etcd{
		Spec: EtcdSpec{
			Members: 0,
			Cpu:     "2",
			Memory:  "2048Mi",
			Storage: "100Gi",
		},
	}

	in.Default()

	assert.Equal(t, 3, in.Spec.Members)
}
