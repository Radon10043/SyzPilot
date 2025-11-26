package generator_test

import (
	"testing"

	"github.com/Radon10043/cloud/src/generator"
)

func TestFindFunction(t *testing.T) {
	db := generator.Database{Path: "data/kernel.db"}
	db.Connect()
	db.GetFunctionCode("pppox_ioctl")
	db.Close()
}
