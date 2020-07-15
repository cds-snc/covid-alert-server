package retrieval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {

	a := 5
	b := 6
	c := -1
	d := -6
	e := 0

	abExpected := a
	acExpected := c
	cdExpected := d
	deExpected := d
	aeExpected := e

	assert.Equal(t, abExpected, min(a, b))
	assert.Equal(t, acExpected, min(a, c))
	assert.Equal(t, cdExpected, min(c, d))
	assert.Equal(t, deExpected, min(d, e))
	assert.Equal(t, aeExpected, min(a, e))
}

func TestTransformRegion(t *testing.T) {

	reg := "302"
	regBadOtherInt := "1233"
	regBadString := "foo"

	ExpectedReg := "CA"
	ExpectedRegBadInt := regBadOtherInt
	ExpectedRegBadString := regBadString

	assert.Equal(t, ExpectedReg, transformRegion(reg))
	assert.Equal(t, ExpectedRegBadInt, transformRegion(regBadOtherInt))
	assert.Equal(t, ExpectedRegBadString, transformRegion(regBadString))
}

// I don't even know where to start with this one
func TestSerializeTo(t *testing.T) {

}
