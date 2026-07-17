package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func moduleRoot() (string, error) {
	cmd := exec.Command("go", "env", "GOMOD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMOD: %w", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" {
		return "", fmt.Errorf("go env GOMOD returned empty string")
	}
	return gomod[:len(gomod)-len("/go.mod")], nil
}

func InstallAPI(ctx context.Context, cl client.Client) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", "tool", "jsonnet", "-e",
		`(import "deploy/lib/api.libsonnet").generate()`,
	)
	cmd.Dir = root
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("jsonnet: %w: %s", err, stderr.String())
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return fmt.Errorf("parsing jsonnet output: %w", err)
	}

	var crdNames []string
	for _, item := range raw {
		obj := &unstructured.Unstructured{}
		if err := obj.UnmarshalJSON(item); err != nil {
			return fmt.Errorf("parsing object: %w", err)
		}
		if err := cl.Create(ctx, obj); err != nil {
			return fmt.Errorf("creating %s %q: %w", obj.GetKind(), obj.GetName(), err)
		}
		if obj.GetKind() == "CustomResourceDefinition" {
			crdNames = append(crdNames, obj.GetName())
		}
	}

	for _, name := range crdNames {
		if err := waitForCRDEstablished(ctx, cl, name); err != nil {
			return err
		}
	}
	return nil
}

func waitForCRDEstablished(ctx context.Context, cl client.Client, name string) error {
	crd := &unstructured.Unstructured{}
	crd.SetAPIVersion("apiextensions.k8s.io/v1")
	crd.SetKind("CustomResourceDefinition")
	return wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		if err := cl.Get(ctx, types.NamespacedName{Name: name}, crd); err != nil {
			return false, err
		}
		conditions, found, err := unstructured.NestedSlice(crd.Object, "status", "conditions")
		if err != nil || !found {
			return false, err
		}
		for _, c := range conditions {
			m, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if m["type"] == "Established" && m["status"] == "True" {
				return true, nil
			}
		}
		return false, nil
	})
}
