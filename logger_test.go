package raopd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLogger(t *testing.T) {
	l := GetLogger("testing")

	assert.NotNil(t, l)
	l.Info().Printf("Testing Info")
	l.Debug().Printf("Testing Debug")
}
