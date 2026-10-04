package installer

import (
	"fmt"
	"os"
	"testing"
)

func TestZZOracleMD5(t *testing.T) {
	data, err := os.ReadFile("/tmp/md5cert.pem")
	if err != nil {
		t.Skip("no fixture")
	}
	fp, err := localCAFingerprint(data)
	fmt.Printf("X509 md5-ca err=%v fplen=%d\n", err != nil, len(fp))
}
