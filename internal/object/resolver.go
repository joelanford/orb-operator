package object

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

type Resolver struct {
	reader client.Reader
}

func NewResolver(reader client.Reader) *Resolver {
	return &Resolver{reader: reader}
}

type Result struct {
	Phases []Phase
	Hash   string
}

type Phase struct {
	Name                string
	CollisionProtection *orbv1alpha1.CollisionProtection
	Objects             []Object
}

type Object struct {
	Obj                 *unstructured.Unstructured
	CollisionProtection *orbv1alpha1.CollisionProtection
	Assertions          []orbv1alpha1.Assertion
}

func (r *Resolver) Resolve(ctx context.Context, specPhases []orbv1alpha1.Phase) (*Result, error) {
	h := sha256.New()
	var phases []Phase

	for _, p := range specPhases {
		rp := Phase{
			Name:                p.Name,
			CollisionProtection: p.CollisionProtection,
		}

		for _, po := range p.Objects {
			raw, err := r.resolveRaw(ctx, p.Name, po)
			if err != nil {
				return nil, err
			}

			obj, err := unmarshalUnstructured(raw)
			if err != nil {
				return nil, fmt.Errorf("phase %q: %w", p.Name, err)
			}
			rp.Objects = append(rp.Objects, Object{
				Obj:                 obj,
				CollisionProtection: po.CollisionProtection,
				Assertions:          po.Assertions,
			})
			h.Write(raw)
		}
		phases = append(phases, rp)
	}

	return &Result{
		Phases: phases,
		Hash:   hex.EncodeToString(h.Sum(nil)),
	}, nil
}

func (r *Resolver) resolveRaw(ctx context.Context, phaseName string, po orbv1alpha1.PhaseObject) ([]byte, error) {
	if po.ObjectRef == nil {
		return po.Object.Raw, nil
	}

	ref := po.ObjectRef
	slice := &orbv1alpha1.ClusterObjectSlice{}
	if err := r.reader.Get(ctx, client.ObjectKey{Name: ref.SliceName}, slice); err != nil {
		return nil, fmt.Errorf("phase %q: fetching slice %q: %w", phaseName, ref.SliceName, err)
	}
	content, ok := lookupInSlice(slice, ref.ObjectKey)
	if !ok {
		return nil, fmt.Errorf(
			"phase %q: object %s %s/%s not found in slice %q",
			phaseName, ref.Kind, ref.Namespace, ref.Name, ref.SliceName)
	}
	if len(content) >= 2 && content[0] == 0x1f && content[1] == 0x8b {
		decompressed, err := decompressGzip(content)
		if err != nil {
			return nil, fmt.Errorf(
				"phase %q: decompress %s %s/%s from slice %q: %w",
				phaseName, ref.Kind, ref.Namespace, ref.Name, ref.SliceName, err)
		}
		return decompressed, nil
	}
	return content, nil
}

func (res *Result) ManagedObjects() []client.Object {
	seen := map[schema.GroupVersionKind]struct{}{}
	var objects []client.Object
	for _, p := range res.Phases {
		for _, o := range p.Objects {
			gvk := o.Obj.GetObjectKind().GroupVersionKind()
			if _, ok := seen[gvk]; ok {
				continue
			}
			seen[gvk] = struct{}{}
			objects = append(objects, o.Obj)
		}
	}
	return objects
}

func (res *Result) VerifyHash(existing string) error {
	if existing == "" {
		return nil
	}
	if existing != res.Hash {
		return fmt.Errorf("resolved content hash mismatch: expected %s, got %s", existing, res.Hash)
	}
	return nil
}

func lookupInSlice(slice *orbv1alpha1.ClusterObjectSlice, key orbv1alpha1.ObjectKey) ([]byte, bool) {
	if slice.ObjectMap != nil {
		content, ok := slice.ObjectMap[key]
		return content, ok
	}
	for _, so := range slice.Objects {
		if so.ObjectKey == key {
			return so.Content, true
		}
	}
	return nil, false
}

func decompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func unmarshalUnstructured(raw []byte) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(raw); err != nil {
		return nil, fmt.Errorf("unmarshalling: %w", err)
	}
	return u, nil
}
