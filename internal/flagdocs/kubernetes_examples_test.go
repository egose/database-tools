package flagdocs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestKubernetesExamplesAreStrictAndSecretBacked(t *testing.T) {
	repoRoot := testRepoRoot(t)
	paths := []string{
		filepath.Join(repoRoot, "examples", "cronjob-archive.yaml"),
		filepath.Join(repoRoot, "examples", "job-copy.yaml"),
		filepath.Join(repoRoot, "examples", "postgres-cronjob-archive.yaml"),
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if strings.Contains(string(content), ":0.3") {
				t.Fatal("example uses stale image tag 0.3")
			}

			for _, document := range splitYAMLDocuments(string(content)) {
				var header kubeHeader
				if err := yaml.Unmarshal([]byte(document), &header); err != nil {
					t.Fatalf("Decode() error = %v", err)
				}
				if header.Kind == "" {
					continue
				}
				validateKubeDocumentStrict(t, []byte(document), header.Kind)
			}
		})
	}
}

func TestJobCopyScriptStopsBeforeRestoreWhenArchiveFails(t *testing.T) {
	repoRoot := testRepoRoot(t)
	content, err := os.ReadFile(filepath.Join(repoRoot, "examples", "job-copy.yaml"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var job jobManifest
	if err := yaml.UnmarshalStrict(content, &job); err != nil {
		t.Fatalf("Job strict schema validation failed: %v", err)
	}
	if len(job.Spec.Template.Spec.Containers) != 1 || len(job.Spec.Template.Spec.Containers[0].Args) != 1 {
		t.Fatalf("job-copy example has unexpected container command shape")
	}
	script := job.Spec.Template.Spec.Containers[0].Args[0]
	if !strings.Contains(script, "--object-name=") {
		t.Fatal("job-copy restore must name the archive object explicitly")
	}

	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.Mkdir(binDir, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "mongo-archive"), []byte("#!/bin/sh\nexit 42\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(mongo-archive) error = %v", err)
	}
	restoreMarker := filepath.Join(tmp, "restore-ran")
	if err := os.WriteFile(filepath.Join(binDir, "mongo-unarchive"), []byte("#!/bin/sh\ntouch \"$RESTORE_MARKER\"\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(mongo-unarchive) error = %v", err)
	}

	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"MONGO_COPY_WORKDIR="+filepath.Join(tmp, "backup"),
		"RESTORE_MARKER="+restoreMarker,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("copy script succeeded with failing archive; output:\n%s", output)
	}
	if _, err := os.Stat(restoreMarker); !os.IsNotExist(err) {
		t.Fatalf("restore command ran after archive failure; stat error = %v", err)
	}
}

func splitYAMLDocuments(content string) []string {
	parts := strings.Split(content, "\n---")
	documents := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		part = strings.TrimPrefix(part, "---")
		part = strings.TrimSpace(part)
		if part != "" {
			documents = append(documents, part)
		}
	}
	return documents
}

func validateKubeDocumentStrict(t *testing.T, content []byte, kind string) {
	t.Helper()
	switch kind {
	case "CronJob":
		var cronJob cronJobManifest
		if err := yaml.UnmarshalStrict(content, &cronJob); err != nil {
			t.Fatalf("CronJob strict schema validation failed: %v", err)
		}
		validatePodSpec(t, cronJob.Spec.JobTemplate.Spec.Template.Spec)
	case "Job":
		var job jobManifest
		if err := yaml.UnmarshalStrict(content, &job); err != nil {
			t.Fatalf("Job strict schema validation failed: %v", err)
		}
		validatePodSpec(t, job.Spec.Template.Spec)
	case "PersistentVolumeClaim":
		var pvc persistentVolumeClaimManifest
		if err := yaml.UnmarshalStrict(content, &pvc); err != nil {
			t.Fatalf("PersistentVolumeClaim strict schema validation failed: %v", err)
		}
	default:
		t.Fatalf("unsupported Kubernetes example kind %q", kind)
	}
}

func validatePodSpec(t *testing.T, spec podSpec) {
	t.Helper()
	containers := append([]containerManifest{}, spec.InitContainers...)
	containers = append(containers, spec.Containers...)
	for _, container := range containers {
		if strings.HasPrefix(container.Image, "ghcr.io/egose/database-tools:") && !strings.HasSuffix(container.Image, ":<latest-version>") {
			t.Fatalf("container %q uses unmaintained image tag %q", container.Name, container.Image)
		}
		for _, envVar := range container.Env {
			if isSensitiveEnvVar(envVar.Name) && envVar.Value != "" {
				t.Fatalf("container %q env %q contains a literal credential; use secretKeyRef", container.Name, envVar.Name)
			}
			if envVar.Value == "" && envVar.ValueFrom == nil {
				t.Fatalf("container %q env %q has neither value nor valueFrom", container.Name, envVar.Name)
			}
		}
	}
}

func isSensitiveEnvVar(name string) bool {
	for _, token := range []string{"URI", "KEY", "PASSWORD", "SECRET", "ACCESS_KEY"} {
		if strings.Contains(name, token) {
			return true
		}
	}
	return false
}

type kubeHeader struct {
	Kind string `yaml:"kind"`
}

type metadataManifest struct {
	Name string `yaml:"name"`
}

type cronJobManifest struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   metadataManifest `yaml:"metadata"`
	Spec       cronJobSpec      `yaml:"spec"`
}

type cronJobSpec struct {
	Schedule          string          `yaml:"schedule"`
	ConcurrencyPolicy string          `yaml:"concurrencyPolicy"`
	JobTemplate       jobTemplateSpec `yaml:"jobTemplate"`
}

type jobTemplateSpec struct {
	Spec jobSpec `yaml:"spec"`
}

type jobManifest struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   metadataManifest `yaml:"metadata"`
	Spec       jobSpec          `yaml:"spec"`
}

type jobSpec struct {
	BackoffLimit int             `yaml:"backoffLimit"`
	Template     podTemplateSpec `yaml:"template"`
}

type podTemplateSpec struct {
	Spec podSpec `yaml:"spec"`
}

type podSpec struct {
	RestartPolicy  string              `yaml:"restartPolicy"`
	InitContainers []containerManifest `yaml:"initContainers"`
	Containers     []containerManifest `yaml:"containers"`
	Volumes        []volumeManifest    `yaml:"volumes"`
}

type containerManifest struct {
	Name            string                `yaml:"name"`
	Image           string                `yaml:"image"`
	ImagePullPolicy string                `yaml:"imagePullPolicy"`
	Command         []string              `yaml:"command"`
	Args            []string              `yaml:"args"`
	Env             []envVarManifest      `yaml:"env"`
	VolumeMounts    []volumeMountManifest `yaml:"volumeMounts"`
}

type envVarManifest struct {
	Name      string                `yaml:"name"`
	Value     string                `yaml:"value"`
	ValueFrom *envVarSourceManifest `yaml:"valueFrom"`
}

type envVarSourceManifest struct {
	SecretKeyRef secretKeyRefManifest `yaml:"secretKeyRef"`
}

type secretKeyRefManifest struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

type volumeMountManifest struct {
	MountPath string `yaml:"mountPath"`
	Name      string `yaml:"name"`
}

type volumeManifest struct {
	Name                  string                       `yaml:"name"`
	PersistentVolumeClaim *persistentVolumeClaimVolume `yaml:"persistentVolumeClaim"`
	EmptyDir              map[string]string            `yaml:"emptyDir"`
}

type persistentVolumeClaimVolume struct {
	ClaimName string `yaml:"claimName"`
}

type persistentVolumeClaimManifest struct {
	APIVersion string                    `yaml:"apiVersion"`
	Kind       string                    `yaml:"kind"`
	Metadata   metadataManifest          `yaml:"metadata"`
	Spec       persistentVolumeClaimSpec `yaml:"spec"`
}

type persistentVolumeClaimSpec struct {
	AccessModes []string                `yaml:"accessModes"`
	Resources   pvcResourceRequirements `yaml:"resources"`
}

type pvcResourceRequirements struct {
	Requests map[string]string `yaml:"requests"`
}
