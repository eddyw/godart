package driver

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed resident.dart
var residentSource string

type residentRequest struct {
	Entry    string            `json:"entry"`
	Output   string            `json:"output"`
	Artifact string            `json:"artifact"`
	Identity string            `json:"identity"`
	Sources  map[string]string `json:"sources"`
	Reset    bool              `json:"reset"`
}

type residentResponse struct {
	Mode         string   `json:"mode"`
	Dependencies []string `json:"dependencies"`
	Error        string   `json:"error"`
}

func dartSDK() (string, error) {
	executable, err := exec.LookPath("dart")
	if err != nil {
		return "", err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	root := filepath.Dir(filepath.Dir(executable))
	for _, sdk := range []string{root, filepath.Join(root, "bin", "cache", "dart-sdk")} {
		info, err := os.Stat(filepath.Join(sdk, "version"))
		if err == nil && !info.IsDir() {
			return filepath.EvalSymlinks(sdk)
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("cannot locate the Dart SDK for %s", executable)
}

func sourceURI(path string) *url.URL {
	return &url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
}

func localURI(uri *url.URL) (string, error) {
	if uri.Scheme != "file" || (uri.Host != "" && uri.Host != "localhost") || uri.RawQuery != "" || uri.Fragment != "" {
		return "", fmt.Errorf("unsupported Dart source URI %q", uri)
	}
	return filepath.Clean(filepath.FromSlash(uri.Path)), nil
}

func residentInputs(output, entry string) (map[string]string, error) {
	config := filepath.Join(output, ".dart_tool", "package_config.json")
	data, err := os.ReadFile(config)
	if err != nil {
		return nil, err
	}
	var packages struct {
		Version  int `json:"configVersion"`
		Packages []struct {
			Root    string `json:"rootUri"`
			Library string `json:"packageUri"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, err
	}
	if packages.Version != 2 {
		return nil, fmt.Errorf("unsupported Dart package configuration")
	}
	roots := map[string]bool{output: true}
	for _, pkg := range packages.Packages {
		root, err := url.Parse(pkg.Root)
		if err != nil {
			return nil, err
		}
		root = sourceURI(config).ResolveReference(root)
		root.Path = strings.TrimRight(root.Path, "/") + "/"
		library, err := url.Parse(pkg.Library)
		if err != nil {
			return nil, err
		}
		path, err := localURI(root.ResolveReference(library))
		if err != nil {
			return nil, err
		}
		roots[path] = true
	}
	paths := map[string]bool{config: true, entry: true}
	for alias := range roots {
		root, err := filepath.EvalSymlinks(alias)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		err = filepath.WalkDir(root, func(path string, item fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if item.IsDir() {
				if path != root && (item.Name() == ".gart" || item.Name() == ".dart_tool") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".dart") {
				resolved, err := filepath.EvalSymlinks(path)
				if err != nil {
					return err
				}
				paths[resolved] = true
				relative, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				paths[filepath.Join(alias, relative)] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	names := make([]string, 0, len(paths))
	for path := range paths {
		names = append(names, path)
	}
	return kernelDependencyHashes(names)
}

func residentDirectory() (string, error) {
	directory := filepath.Join("/tmp", fmt.Sprintf("gart-%d", os.Getuid()))
	if err := os.Mkdir(directory, 0700); err != nil && !os.IsExist(err) {
		return "", err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 || info.Sys().(*syscall.Stat_t).Uid != uint32(os.Getuid()) {
		return "", fmt.Errorf("resident directory must be private and owned by the current user: %s", directory)
	}
	// Unix socket paths have a small platform limit, even when TMPDIR is long.
	namespace := sha256.Sum256([]byte(os.TempDir()))
	directory = filepath.Join(directory, hex.EncodeToString(namespace[:8]))
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	return directory, nil
}

func startResident(directory, socket, sdk string) (*exec.Cmd, error) {
	script := filepath.Join(directory, "service.dart")
	if err := os.WriteFile(script, []byte(residentSource), 0600); err != nil {
		return nil, err
	}
	log, err := os.OpenFile(filepath.Join(directory, "service.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer log.Close()
	cmd := exec.Command(filepath.Join(sdk, "bin", "dart"), "--disable-dart-dev", script, socket, sdk)
	cmd.Dir = directory
	cmd.Stdin = nil
	cmd.Stdout, cmd.Stderr = log, log
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() { _ = cmd.Wait() }()
	return cmd, nil
}

func residentCompile(sdk, identity string, request residentRequest) (residentResponse, error) {
	var response residentResponse
	directory, err := residentDirectory()
	if err != nil {
		return response, err
	}
	digest := sha256.Sum256([]byte(identity))
	name := hex.EncodeToString(digest[:16])
	socket := filepath.Join(directory, name+".sock")
	// The startup lock serializes stale socket replacement across CLI processes.
	lock, err := os.OpenFile(filepath.Join(directory, name+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return response, err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return response, err
	}
	conn, err := net.DialTimeout("unix", socket, time.Second)
	if err != nil {
		if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
			return response, err
		}
		serviceDir := filepath.Join(directory, name)
		if err := os.MkdirAll(serviceDir, 0700); err != nil {
			return response, err
		}
		child, err := startResident(serviceDir, socket, sdk)
		if err != nil {
			return response, err
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			conn, err = net.DialTimeout("unix", socket, time.Second)
			if err == nil {
				break
			}
			if time.Now().After(deadline) {
				_ = child.Process.Kill()
				log, _ := os.ReadFile(filepath.Join(serviceDir, "service.log"))
				return response, fmt.Errorf("resident compiler startup: %w\n%s", err, log)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(125 * time.Second)); err != nil {
		return response, err
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return response, err
	}
	if err := json.NewDecoder(io.LimitReader(conn, 16<<20)).Decode(&response); err != nil {
		return response, fmt.Errorf("resident compiler response: %w", err)
	}
	if response.Error != "" {
		return response, fmt.Errorf("resident compile: %s", response.Error)
	}
	return response, nil
}
