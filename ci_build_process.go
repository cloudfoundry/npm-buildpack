package npminstall

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

type CIBuildProcess struct {
	executable Executable
	summer     Summer
	logger     scribe.Logger
}

func NewCIBuildProcess(executable Executable, summer Summer, logger scribe.Logger) CIBuildProcess {
	return CIBuildProcess{
		executable: executable,
		summer:     summer,
		logger:     logger,
	}
}

func (r CIBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}) (bool, string, error) {
	sum, err := r.summer.Sum(filepath.Join(workingDir, "package-lock.json"))
	if err != nil {
		return false, "", err
	}

	cacheSha, ok := metadata["cache_sha"].(string)
	if !ok || sum != cacheSha {
		return true, sum, nil
	}

	return false, "", nil
}

func (r CIBuildProcess) Run(modulesDir, cacheDir, workingDir string) error {
	err := os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)
	if err != nil {
		return err
	}

	err = fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(modulesDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(modulesDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(nil)
	args := []string{"ci", "--unsafe-perm", "--cache", cacheDir}

	r.logger.Subprocess("Running 'npm %s'", strings.Join(args, " "))
	err = r.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
		Env:    append(os.Environ(), "NPM_CONFIG_PRODUCTION=true", "NPM_CONFIG_LOGLEVEL=error"),
	})
	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("npm ci failed: %w", err)
	}

	return nil
}
