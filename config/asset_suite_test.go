package config

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAsset(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Asset Suite")
}
