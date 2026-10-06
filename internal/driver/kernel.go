package driver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	compatibility "github.com/eddyw/godart/internal/sdk"
)

// Kernel compilation defaults to product mode. Source execution uses non-product constants.
var kernelFlags = []string{"-Ddart.vm.product=false"}

type kernelReceipt struct {
	Checksum     string
	Identity     string
	Dependencies map[string]string
	Hash         string
}

func (receipt kernelReceipt) checksum() string {
	receipt.Checksum = ""
	data, _ := json.Marshal(receipt)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func kernelDependencyHashes(paths []string) (map[string]string, error) {
	hashes := make(map[string]string, len(paths))
	for _, path := range paths {
		hash, err := fileDigest(path)
		if err != nil {
			return nil, err
		}
		hashes[path] = hash
	}
	return hashes, nil
}

func sameDependencies(hashes map[string]string) bool {
	if len(hashes) == 0 {
		return false
	}
	for path, want := range hashes {
		got, err := fileDigest(path)
		if err != nil || got != want {
			return false
		}
	}
	return true
}

func dartIdentity() (string, string, error) {
	sdk, err := dartSDK()
	if err != nil {
		return "", "", err
	}
	version, err := os.ReadFile(filepath.Join(sdk, "version"))
	if err != nil {
		return "", "", err
	}
	versions, err := compatibility.Read()
	if err != nil {
		return "", "", err
	}
	if !slices.Contains(versions.Dart.Resident, strings.TrimSpace(string(version))) {
		return "", "", fmt.Errorf("resident compilation requires Dart %s, found %s", strings.Join(versions.Dart.Resident, " or "), strings.TrimSpace(string(version)))
	}
	paths := []string{}
	for _, name := range []string{"version", "revision", "bin/dart", "bin/dartvm", "bin/dartaotruntime", "bin/snapshots/frontend_server_aot.dart.snapshot", "lib/_internal/vm_platform_strong.dill", "lib/libraries.json"} {
		path := filepath.Join(sdk, filepath.FromSlash(name))
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		}
	}
	hashes, err := kernelDependencyHashes(paths)
	if err != nil {
		return "", "", err
	}
	data, err := json.Marshal(struct {
		SDK     string
		Flags   []string
		Files   map[string]string
		Service string
	}{sdk, kernelFlags, hashes, residentSource})
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(data)
	return sdk, hex.EncodeToString(sum[:]), nil
}

func (opt Options) kernel(entry string) (string, error) {
	release, err := opt.lockOutput()
	if err != nil {
		return "", err
	}
	defer release()
	start := time.Now()
	output, err := filepath.Abs(opt.Out)
	if err != nil {
		return "", err
	}
	output, err = filepath.EvalSymlinks(output)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.EvalSymlinks(filepath.Join(output, entry))
	if err != nil {
		return "", err
	}
	sdk, identity, err := dartIdentity()
	if err != nil {
		return "", err
	}
	name := sha256.Sum256([]byte(absolute))
	directory := filepath.Join(output, ".gart", "kernel")
	base := filepath.Join(directory, hex.EncodeToString(name[:]))
	kernel, manifest := base+".dill", base+".json"
	var receipt kernelReceipt
	data, readErr := os.ReadFile(manifest)
	if !opt.NoCache && readErr == nil && json.Unmarshal(data, &receipt) == nil && receipt.Identity == identity && receipt.Checksum == receipt.checksum() && sameDependencies(receipt.Dependencies) {
		hash, err := fileDigest(kernel)
		if err == nil && hash == receipt.Hash {
			opt.trace("kernel-cache", start)
			return kernel, nil
		}
	}
	opt.trace("kernel-cache", start)
	start = time.Now()
	defer func() { opt.trace("kernel-compile", start) }()
	sources, err := residentInputs(output, absolute)
	if err != nil {
		return "", err
	}
	assets := filepath.Join(output, ".dart_tool", "native_assets.yaml")
	if _, err := os.Stat(assets); err == nil {
		hash, err := fileDigest(assets)
		if err != nil {
			return "", err
		}
		sources[assets] = hash
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(directory, "compile-*.dill")
	if err != nil {
		return "", err
	}
	temporary := temp.Name()
	if err := temp.Close(); err != nil {
		return "", err
	}
	defer os.Remove(temporary)
	config := filepath.Join(output, ".dart_tool", "package_config.json")
	response, err := residentCompile(sdk, identity, residentRequest{
		Entry: absolute, Output: output, Artifact: temporary, Identity: identity + sources[config], Sources: sources, Reset: opt.NoCache,
	})
	if err != nil {
		return "", err
	}
	if response.Mode != "cold" && response.Mode != "incremental" {
		return "", fmt.Errorf("invalid resident compiler mode %q", response.Mode)
	}
	opt.trace("kernel-"+response.Mode, start)
	if !sameDependencies(sources) {
		return "", fmt.Errorf("compiler inputs changed during compilation; retry the request")
	}
	_, after, err := dartIdentity()
	if err != nil {
		return "", err
	}
	if after != identity {
		return "", fmt.Errorf("Dart SDK changed during compilation; retry the request")
	}
	receipt = kernelReceipt{Identity: identity, Dependencies: map[string]string{}}
	for _, path := range append(response.Dependencies, absolute, config) {
		hash, ok := sources[path]
		if !ok {
			return "", fmt.Errorf("compiler dependency outside the source snapshot: %s", path)
		}
		receipt.Dependencies[path] = hash
	}
	receipt.Hash, err = fileDigest(temporary)
	if err != nil {
		return "", err
	}
	receipt.Checksum = receipt.checksum()
	data, err = json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	if err := os.Rename(temporary, kernel); err != nil {
		return "", err
	}
	if err := os.WriteFile(temporary, data, 0644); err != nil {
		return "", err
	}
	if err := os.Rename(temporary, manifest); err != nil {
		return "", err
	}
	return kernel, nil
}
