package publish

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	credentialObjectLimit = 10000
	credentialObjectBytes = 4 << 20
	credentialTotalBytes  = 32 << 20
)

// Scan from the admitted base even when retrying a previously published tip.
// Only that base is trusted; a prior candidate must not exempt worker objects.
func (g *repository) checkCredentials(ctx context.Context, r Request) error {
	out, err := g.run(ctx, "rev-list", "--objects", "--no-object-names", r.Commit, "^"+r.BaseSHA)
	if err != nil {
		return err
	}
	objects := strings.Fields(string(out))
	if len(objects) > credentialObjectLimit {
		return errors.New("candidate exceeds credential object limit")
	}
	if len(objects) == 0 {
		return nil
	}
	scanctx, cancel := context.WithCancel(ctx)
	defer cancel()
	command := g.command(scanctx, "cat-file", "--batch")
	command.Stdin = strings.NewReader(strings.Join(objects, "\n") + "\n")
	output, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	if err = command.Start(); err != nil {
		return errors.New("publication credential scan failed")
	}
	scanErr := scanCredentials(bufio.NewReader(output), objects, r.ProtectedCredentials)
	if scanErr != nil {
		cancel()
	}
	waitErr := command.Wait()
	return credentialScanResult(ctx, scanErr, waitErr)
}

func credentialScanResult(ctx context.Context, scanErr, waitErr error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if scanErr != nil {
		return scanErr
	}
	if waitErr != nil {
		return errors.New("publication credential scan failed")
	}
	return nil
}

func scanCredentials(reader *bufio.Reader, objects, secrets []string) error {
	var total int64
	for _, object := range objects {
		size, err := scanCredentialObject(reader, object, credentialTotalBytes-total, secrets)
		if err != nil {
			return err
		}
		total += size
	}
	return nil
}

func credentialObjectHeader(reader *bufio.Reader, object string, remaining int64) (string, int64, error) {
	header, err := reader.ReadSlice('\n')
	if err != nil {
		return "", 0, errors.New("invalid publication object header")
	}
	fields := strings.Fields(string(header))
	if len(fields) != 3 || fields[0] != object {
		return "", 0, errors.New("publication object identity differs")
	}
	switch fields[1] {
	case "blob", "tree", "commit", "tag":
	default:
		return "", 0, errors.New("invalid publication object type")
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return "", 0, errors.New("invalid publication object size")
	}
	if !credentialObjectSizeAllowed(size, remaining) {
		return "", 0, errors.New("candidate exceeds decoded credential scan limit")
	}
	return fields[1], size, nil
}

func credentialObjectSizeAllowed(size, remaining int64) bool {
	return size >= 0 && size <= credentialObjectBytes && size <= remaining
}

func scanCredentialObject(reader *bufio.Reader, object string, remaining int64, secrets []string) (int64, error) {
	kind, size, err := credentialObjectHeader(reader, object, remaining)
	if err != nil {
		return 0, err
	}
	data := make([]byte, int(size)+1)
	if _, err = io.ReadFull(reader, data); err != nil || data[len(data)-1] != '\n' {
		return 0, errors.New("invalid publication object content")
	}
	data = data[:len(data)-1]
	content := sha1.New()
	_, _ = fmt.Fprintf(content, "%s %d\x00", kind, size)
	_, _ = content.Write(data)
	if fmt.Sprintf("%x", content.Sum(nil)) != object {
		return 0, errors.New("publication object content differs from identity")
	}
	for _, secret := range secrets {
		if secret != "" && bytes.Contains(data, []byte(secret)) {
			return 0, errors.New("candidate contains protected credential material")
		}
	}
	return size, nil
}
