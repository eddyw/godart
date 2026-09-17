package driver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Kernel compilation defaults to product mode. Source execution uses non-product constants.
var kernelFlags = []string{"-Ddart.vm.product=false"}

type kernelReceipt struct {
	Identity     string
	Dependencies map[string]string
	Hash         string
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

func dartIdentity() (string, error) {
	executable, err := exec.LookPath("dart")
	if err != nil {
		return "", err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	version, err := exec.Command(executable, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("dart --version: %w", err)
	}
	paths := []string{executable}
	sdk := filepath.Dir(filepath.Dir(executable))
	for _, name := range []string{"version", "revision", "bin/dartvm", "bin/dartaotruntime", "bin/snapshots/gen_kernel_aot.dart.snapshot", "lib/_internal/vm_platform_strong.dill"} {
		path := filepath.Join(sdk, filepath.FromSlash(name))
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		}
	}
	hashes, err := kernelDependencyHashes(paths)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(struct {
		Version string
		Flags   []string
		Files   map[string]string
	}{string(version), kernelFlags, hashes})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (opt Options) kernel(entry string) (string, error) {
	start := time.Now()
	absolute, err := filepath.Abs(filepath.Join(opt.Out, entry))
	if err != nil {
		return "", err
	}
	identity, err := dartIdentity()
	if err != nil {
		return "", err
	}
	name := sha256.Sum256([]byte(absolute))
	directory, err := filepath.Abs(filepath.Join(opt.Out, ".gart", "kernel"))
	if err != nil {
		return "", err
	}
	base := filepath.Join(directory, hex.EncodeToString(name[:]))
	kernel := base + ".dill"
	manifest := base + ".json"
	var receipt kernelReceipt
	data, readErr := os.ReadFile(manifest)
	if !opt.NoCache && readErr == nil && json.Unmarshal(data, &receipt) == nil && receipt.Identity == identity && sameDependencies(receipt.Dependencies) {
		hash, err := fileDigest(kernel)
		if err == nil && hash == receipt.Hash {
			opt.trace("kernel-cache", start)
			return kernel, nil
		}
	}
	opt.trace("kernel-cache", start)
	start = time.Now()
	defer func() { opt.trace("kernel-compile", start) }()
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
	depfile := temporary + ".d"
	defer os.Remove(depfile)
	args := append([]string{"compile", "kernel"}, kernelFlags...)
	args = append(args, "--depfile="+depfile, "-o", temporary, absolute)
	cmd := exec.Command("dart", args...)
	cmd.Dir = opt.Out
	if text, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("dart compile kernel %s: %w\n%s", entry, err, text)
	}
	data, err = os.ReadFile(depfile)
	if err != nil {
		return "", err
	}
	paths, err := kernelDependencies(string(data))
	if err != nil {
		return "", err
	}
	for i, path := range paths {
		if !filepath.IsAbs(path) {
			paths[i] = filepath.Join(opt.Out, path)
		}
	}
	paths = append(paths, absolute, filepath.Join(opt.Out, ".dart_tool", "package_config.json"))
	for i, path := range paths {
		paths[i], err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}
	slices.Sort(paths)
	paths = slices.Compact(paths)
	receipt.Dependencies, err = kernelDependencyHashes(paths)
	if err != nil {
		return "", err
	}
	cacheable := true
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		if info.ModTime().After(start) {
			cacheable = false
		}
	}
	receipt.Identity = identity
	receipt.Hash, err = fileDigest(temporary)
	if err != nil {
		return "", err
	}
	data, err = json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	if err := os.Rename(temporary, kernel); err != nil {
		return "", err
	}
	if !cacheable {
		if err := os.Remove(manifest); err != nil && !os.IsNotExist(err) {
			return "", err
		}
		return kernel, nil
	}
	if err := os.WriteFile(depfile, data, 0644); err != nil {
		return "", err
	}
	if err := os.Rename(depfile, manifest); err != nil {
		return "", err
	}
	return kernel, nil
}

// Dart emits Ninja depfiles. Escaped spaces belong to a path, not to its separator.
func kernelDependencies(text string) ([]string, error) {
	var words []string
	var word strings.Builder
	target := true
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '\\' && i+1 < len(text) {
			i++
			if text[i] != '\n' {
				word.WriteByte(text[i])
			}
			continue
		}
		if c == '$' && i+1 < len(text) && text[i+1] == '$' {
			word.WriteByte('$')
			i++
			continue
		}
		if target && c == ':' && i+1 < len(text) && (text[i+1] == ' ' || text[i+1] == '\n') {
			word.Reset()
			target = false
			continue
		}
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			if word.Len() > 0 {
				words = append(words, word.String())
				word.Reset()
			}
		} else {
			word.WriteByte(c)
		}
	}
	if word.Len() > 0 {
		words = append(words, word.String())
	}
	if target || len(words) == 0 {
		return nil, fmt.Errorf("invalid Dart kernel dependency file")
	}
	return words, nil
}
