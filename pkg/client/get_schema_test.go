package client

import (
	"testing"

	pkg "kcl-lang.io/kpm/pkg/package"
)

func TestGetSchemaType(t *testing.T) {
	testFunc := func(t *testing.T, kpmcli *KpmClient) {
		kmod, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath("TODO: the test path"),
		)
		if err != nil {
			t.Fatal(err)
		}

		err = kpmcli.GetSchemaType(
			WithKmod(kmod),
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "TestGetSchemaType", TestFunc: testFunc}})
}
