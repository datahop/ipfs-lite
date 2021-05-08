package datahop

import (
	"os"
	"testing"

	ipfslite "github.com/datahop/ipfs-lite"
)

func TestContentLength(t *testing.T) {
	root := "/tmp" + string(os.PathSeparator) + ipfslite.Root
	err := Init(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = DiskUsage()
	if err != nil {
		t.Fatal(err)
	}
}
