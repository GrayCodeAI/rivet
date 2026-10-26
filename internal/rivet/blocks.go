package rivet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

type Block struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Status struct {
	Block    Block  `json:"block"`
	Verified bool   `json:"verified"`
	Reason   string `json:"reason,omitempty"`
}

type receipt struct {
	Hashes    [3]string `json:"hashes"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	CheckedAt time.Time `json:"checked_at"`
}

type Store struct {
	Project string
	LogDir  string
}

func (s Store) Root() (string, error) {
	root, err := filepath.Abs(s.Project)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project is not a directory: %s", root)
	}
	return root, nil
}

func (s Store) Base() (string, error) {
	root, err := s.Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".rivet"), nil
}

func (s Store) Init() error {
	base, err := s.Base()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(base, "blocks"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(base, "logs"), 0700); err != nil {
		return err
	}
	ignore := filepath.Join(base, ".gitignore")
	f, err := os.OpenFile(ignore, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(f, "blocks/*/verified.json\nlogs/\n")
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func (s Store) blockDir(name string) (string, error) {
	if !validName.MatchString(name) {
		return "", fmt.Errorf("invalid block name %q", name)
	}
	base, err := s.Base()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "blocks", name), nil
}

func regular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file", path)
	}
	return os.ReadFile(path)
}

func (s Store) Add(name, description, script, check string) error {
	dir, err := s.blockDir(name)
	if err != nil {
		return err
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return errors.New("description is required")
	}
	runBytes, err := regular(script)
	if err != nil {
		return fmt.Errorf("script: %w", err)
	}
	checkBytes, err := regular(check)
	if err != nil {
		return fmt.Errorf("check: %w", err)
	}
	if len(runBytes) == 0 || len(checkBytes) == 0 {
		return errors.New("script and check must be nonempty")
	}
	if err := s.Init(); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(dir)
		}
	}()
	manifest, err := json.MarshalIndent(Block{Name: name, Description: description}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "block.json"), append(manifest, '\n'), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "run.sh"), runBytes, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "verify.sh"), checkBytes, 0755); err != nil {
		return err
	}
	complete = true
	return nil
}

func (s Store) Show(name string) (Status, error) {
	dir, err := s.blockDir(name)
	if err != nil {
		return Status{}, err
	}
	data, err := regular(filepath.Join(dir, "block.json"))
	if err != nil {
		return Status{}, err
	}
	var block Block
	if err := json.Unmarshal(data, &block); err != nil {
		return Status{}, err
	}
	if block.Name != name || strings.TrimSpace(block.Description) == "" {
		return Status{}, errors.New("invalid block manifest")
	}
	status := Status{Block: block}
	if err := s.requireVerified(name); err != nil {
		status.Reason = err.Error()
	} else {
		status.Verified = true
	}
	return status, nil
}

func (s Store) List() ([]Status, error) {
	base, err := s.Base()
	if err != nil {
		return nil, err
	}
	dirs, err := os.ReadDir(filepath.Join(base, "blocks"))
	if errors.Is(err, os.ErrNotExist) {
		return []Status{}, nil
	}
	if err != nil {
		return nil, err
	}
	statuses := make([]Status, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		status, err := s.Show(dir.Name())
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Block.Name < statuses[j].Block.Name })
	return statuses, nil
}

func hashes(dir string) ([3]string, error) {
	var result [3]string
	for i, name := range []string{"block.json", "run.sh", "verify.sh"} {
		data, err := regular(filepath.Join(dir, name))
		if err != nil {
			return result, err
		}
		sum := sha256.Sum256(data)
		result[i] = hex.EncodeToString(sum[:])
	}
	return result, nil
}

func (s Store) Verify(ctx context.Context, name string, output io.Writer) error {
	dir, err := s.blockDir(name)
	if err != nil {
		return err
	}
	if _, err := s.Show(name); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(dir, "verified.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	before, err := hashes(dir)
	if err != nil {
		return err
	}
	root, err := s.Root()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "sh", filepath.Join(dir, "verify.sh"))
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "RIVET_SCRIPT="+filepath.Join(dir, "run.sh"), "RIVET_PROJECT_ROOT="+root)
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}
	after, err := hashes(dir)
	if err != nil {
		return err
	}
	if before != after {
		return errors.New("block changed during verification")
	}
	rec := receipt{Hashes: after, OS: runtime.GOOS, Arch: runtime.GOARCH, CheckedAt: time.Now().UTC()}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "verified.json"), append(data, '\n'), 0644)
}

func (s Store) requireVerified(name string) error {
	dir, err := s.blockDir(name)
	if err != nil {
		return err
	}
	data, err := regular(filepath.Join(dir, "verified.json"))
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("block is unverified; run rivet verify")
	}
	if err != nil {
		return err
	}
	var rec receipt
	if err := json.Unmarshal(data, &rec); err != nil {
		return err
	}
	if rec.OS != runtime.GOOS || rec.Arch != runtime.GOARCH {
		return errors.New("block verified on another platform")
	}
	current, err := hashes(dir)
	if err != nil {
		return err
	}
	if current != rec.Hashes {
		return errors.New("block changed since verification; run rivet verify")
	}
	return nil
}

func (s Store) BlockCommand(ctx context.Context, name string, args []string) (*exec.Cmd, error) {
	if err := s.requireVerified(name); err != nil {
		return nil, err
	}
	dir, err := s.blockDir(name)
	if err != nil {
		return nil, err
	}
	root, err := s.Root()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "sh", append([]string{filepath.Join(dir, "run.sh")}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "RIVET_BLOCK_DIR="+dir, "RIVET_PROJECT_ROOT="+root)
	return cmd, nil
}
