package exif

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ReadMetadata extracts ALL metadata from imageData via an exiftool shell-out
// reading from stdin — the image never touches disk. Groups are family-1
// (-g1: IFD0, ExifIFD, Canon, ...) so a viewer can render one section per
// group; -a keeps duplicate tags, -u includes unknown ones.
// The InjectMetadata semaphore also bounds these processes.
func ReadMetadata(ctx context.Context, imageData []byte) (map[string]any, error) {
	slot := currentSem()
	select {
	case slot <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-slot }()

	cmd := exec.CommandContext(ctx, "exiftool", "-j", "-a", "-u", "-g1", "-")
	cmd.Stdin = bytes.NewReader(imageData)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("exiftool: %w", err)
	}

	var results []map[string]any
	if err := json.Unmarshal(out, &results); err != nil {
		return nil, fmt.Errorf("exiftool output: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("exiftool returned no metadata")
	}
	meta := results[0]
	delete(meta, "SourceFile") // always "-" for stdin; noise for the caller
	return meta, nil
}
