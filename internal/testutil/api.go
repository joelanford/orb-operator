package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	// After creating all objects, wait for each CRD resource to be fully
	// served by the API server. When MutatingAdmissionPolicies are
	// installed before CRDs (the production ordering), the API server's
	// MAP admission dispatcher needs time to sync its type resolver with
	// newly-created CRDs. During this window, CREATE requests return
	// ServiceUnavailable even though the CRD Established condition is True.
	// A dry-run CREATE probes the full admission chain without persisting.
	for _, name := range crdNames {
		if err := waitForCRDServable(ctx, cl, name); err != nil {
			return err
		}
	}
	return nil
}

func waitForCRDServable(ctx context.Context, cl client.Client, name string) error {
	crd := &unstructured.Unstructured{}
	crd.SetAPIVersion("apiextensions.k8s.io/v1")
	crd.SetKind("CustomResourceDefinition")
	if err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 10*time.Second, true, func(ctx context.Context) (bool, error) {
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
	}); err != nil {
		return err
	}

	group, _, _ := unstructured.NestedString(crd.Object, "spec", "group")
	kind, _, _ := unstructured.NestedString(crd.Object, "spec", "names", "kind")
	versionsRaw, _, _ := unstructured.NestedSlice(crd.Object, "spec", "versions")
	for _, v := range versionsRaw {
		vm, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		served, _, _ := unstructured.NestedBool(vm, "served")
		if !served {
			continue
		}
		version := vm["name"].(string)
		gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
		if err := waitForDryRunCreate(ctx, cl, gvk); err != nil {
			return fmt.Errorf("waiting for %s to be servable: %w", gvk, err)
		}
	}
	return nil
}

func waitForDryRunCreate(ctx context.Context, cl client.Client, gvk schema.GroupVersionKind) error {
	return wait.PollUntilContextTimeout(ctx, 200*time.Millisecond, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		obj.SetName("dry-run-probe")
		err := cl.Create(ctx, obj, client.DryRunAll)
		if errors.IsServiceUnavailable(err) {
			return false, nil
		}
		return true, nil
	})
}
