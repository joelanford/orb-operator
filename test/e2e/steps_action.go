package e2e

import (
	"context"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func registerActionSteps(sc *godog.ScenarioContext, tc *testContext) {
	sc.Step(`^the COSR is created$`, tc.theCOSRIsCreated)
	sc.Step(`^the COSR is created and becomes Available$`, tc.theCOSRIsCreatedAndBecomesAvailable)
	sc.Step(`^the ConfigMap "([^"]*)" is deleted$`, tc.theConfigMapIsDeleted)
	sc.Step(`^the COSR lifecycleState is set to "([^"]*)"$`, tc.theCOSRLifecycleStateIsSetTo)
	sc.Step(`^setting the COSR lifecycleState to "([^"]*)" should fail$`, tc.settingCOSRLifecycleStateShouldFail)
	sc.Step(`^all phases complete successfully$`, tc.allPhasesComplete)
	sc.Step(`^a COSR with group "([^"]*)" and revision (\d+) is created$`, tc.aNewCOSRIsCreated)
	sc.Step(`^the new COSR is created$`, tc.theNewCOSRIsCreated)
	sc.Step(`^the new COSR is created and becomes Available$`, tc.theNewCOSRIsCreatedAndBecomesAvailable)
	sc.Step(`^revision (\d+) is archived$`, tc.revisionIsArchived)
	sc.Step(`^creating the COSR should fail$`, tc.creatingTheCOSRShouldFail)
	sc.Step(`^creating a COSR with zero phases should fail$`, tc.creatingCOSRWithZeroPhasesShouldFail)
	sc.Step(`^creating a COSR with a phase with zero objects should fail$`, tc.creatingCOSRWithZeroObjectsShouldFail)
	sc.Step(`^updating the COSR group should fail$`, tc.updatingCOSRGroupShouldFail)
	sc.Step(`^updating the COSR revision should fail$`, tc.updatingCOSRRevisionShouldFail)
	sc.Step(`^updating the COSR phases should fail$`, tc.updatingCOSRPhasesShouldFail)
	sc.Step(`^updating the COSR collisionProtection should fail$`, tc.updatingCOSRCollisionProtectionShouldFail)
	sc.Step(`^creating a COSR with revision 0 should fail$`, tc.creatingCOSRWithRevisionZeroShouldFail)
	sc.Step(`^creating a COSR with unset lifecycleState should fail$`, tc.creatingCOSRWithUnsetLifecycleStateShouldFail)
	sc.Step(`^creating a COSR with unknown lifecycleState should fail$`, tc.creatingCOSRWithUnknownLifecycleStateShouldFail)
	sc.Step(`^creating a COSR with a group name of exactly 52 characters should succeed$`, tc.creatingCOSRWithExact52CharGroupShouldSucceed)
	sc.Step(`^creating a COSR with a group name longer than 52 characters should fail$`, tc.creatingCOSRWithLongGroupShouldFail)
	sc.Step(`^the COSR is deleted with cascade foreground$`, tc.theCOSRIsDeletedWithCascadeForeground)
	sc.Step(`^the COSR is deleted with cascade background$`, tc.theCOSRIsDeletedWithCascadeBackground)
	sc.Step(`^the COSR is deleted with cascade orphan$`, tc.theCOSRIsDeletedWithCascadeOrphan)
	sc.Step(`^the CRD "([^"]*)" is deleted$`, tc.theCRDIsDeleted)
	sc.Step(`^the ConfigMap "([^"]*)" field "([^"]*)" is set to "([^"]*)"$`, tc.theConfigMapFieldIsSetTo)
	sc.Step(`^the ConfigMap "([^"]*)" is recreated by the controller$`, tc.theConfigMapIsRecreatedByController)

	sc.Step(`^the COSR with group "([^"]*)" and revision (\d+) lifecycleState is set to "([^"]*)"$`, tc.theCOSRInGroupLifecycleStateIsSetTo)

	// COS action steps
	sc.Step(`^the COS is created$`, tc.theCOSIsCreated)
	sc.Step(`^the COS template spec is updated with a ConfigMap "([^"]*)" in phase "([^"]*)"$`, tc.theCOSTemplateSpecIsUpdated)
	sc.Step(`^the COS template spec is updated with a gated ConfigMap "([^"]*)" in phase "([^"]*)"$`, tc.theCOSTemplateSpecIsUpdatedWithGatedConfigMap)
	sc.Step(`^the COS template label "([^"]*)" is updated to "([^"]*)"$`, tc.theCOSTemplateLabelIsUpdated)
	sc.Step(`^the COS "([^"]*)" is deleted$`, tc.theCOSIsDeleted)
	sc.Step(`^the COS "([^"]*)" is deleted with cascade orphan$`, tc.theCOSIsDeletedWithCascadeOrphan)
	sc.Step(`^the COS "([^"]*)" label "([^"]*)" is set to "([^"]*)"$`, tc.theCOSLabelIsSetTo)
	sc.Step(`^the COS "([^"]*)" revisionHistoryLimit is set to (\d+)$`, tc.theCOSRevisionHistoryLimitIsSetTo)
	sc.Step(`^creating a COS with a name of exactly 52 characters should succeed$`, tc.creatingCOSWithExact52CharNameShouldSucceed)
	sc.Step(`^creating a COS with a name longer than 52 characters should fail$`, tc.creatingCOSWithLongNameShouldFail)
}

func (tc *testContext) theCOSRIsCreated() error {
	return tc.createCOSR(context.Background())
}

func (tc *testContext) theCOSRIsCreatedAndBecomesAvailable() error {
	if err := tc.createCOSR(context.Background()); err != nil {
		return err
	}
	return tc.pollForCOSRCondition(context.Background(), tc.lastCreatedCOSRName(), "Available", metav1.ConditionTrue)
}

func (tc *testContext) theConfigMapIsDeleted(name string) error {
	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: tc.namespace, Name: name}
	if err := tc.client.Get(context.Background(), key, cm); err != nil {
		return fmt.Errorf("getting ConfigMap %q: %w", name, err)
	}
	return tc.client.Delete(context.Background(), cm)
}

func (tc *testContext) theCOSRIsDeletedWithCascadeForeground() error {
	return tc.deleteCOSRWithPropagation(metav1.DeletePropagationForeground)
}

func (tc *testContext) theCOSRIsDeletedWithCascadeBackground() error {
	return tc.deleteCOSRWithPropagation(metav1.DeletePropagationBackground)
}

func (tc *testContext) deleteCOSRWithPropagation(policy metav1.DeletionPropagation) error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	return tc.client.Delete(context.Background(), cosr, &client.DeleteOptions{
		PropagationPolicy: &policy,
	})
}

func (tc *testContext) theCOSRIsDeletedWithCascadeOrphan() error {
	return tc.deleteCOSRWithPropagation(metav1.DeletePropagationOrphan)
}

func (tc *testContext) theCRDIsDeleted(name string) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	key := types.NamespacedName{Name: name + ".e2e.orb.dev"}
	if err := tc.client.Get(context.Background(), key, crd); err != nil {
		return fmt.Errorf("getting CRD %q: %w", name, err)
	}
	return tc.client.Delete(context.Background(), crd)
}

func (tc *testContext) theCOSRLifecycleStateIsSetTo(state string) error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	cosr.Spec.LifecycleState = orbv1alpha1.LifecycleState(state)
	return tc.client.Update(context.Background(), cosr)
}

func (tc *testContext) settingCOSRLifecycleStateShouldFail(state string) error {
	err := tc.theCOSRLifecycleStateIsSetTo(state)
	if err == nil {
		return fmt.Errorf("expected setting lifecycleState to %q to fail, but it succeeded", state)
	}
	return nil
}

func (tc *testContext) creatingTheCOSRShouldFail() error {
	err := tc.createCOSR(context.Background())
	if err == nil {
		return fmt.Errorf("expected COSR creation to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) creatingCOSRWithZeroPhasesShouldFail() error {
	tc.resetCOSRBuilder("zero-phases", 1)
	err := tc.createCOSR(context.Background())
	if err == nil {
		return fmt.Errorf("expected COSR with zero phases to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) creatingCOSRWithZeroObjectsShouldFail() error {
	tc.resetCOSRBuilder("zero-objects", 1)
	tc.addPhase("empty")
	err := tc.createCOSR(context.Background())
	if err == nil {
		return fmt.Errorf("expected COSR with zero objects in phase to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) updatingCOSRGroupShouldFail() error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	cosr.Spec.Group = cosr.Spec.Group + "-changed"
	err := tc.client.Update(context.Background(), cosr)
	if err == nil {
		return fmt.Errorf("expected updating group to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) updatingCOSRRevisionShouldFail() error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	cosr.Spec.Revision = cosr.Spec.Revision + 1
	err := tc.client.Update(context.Background(), cosr)
	if err == nil {
		return fmt.Errorf("expected updating revision to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) updatingCOSRPhasesShouldFail() error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	cosr.Spec.Phases = append(cosr.Spec.Phases, orbv1alpha1.Phase{Name: "injected"})
	err := tc.client.Update(context.Background(), cosr)
	if err == nil {
		return fmt.Errorf("expected updating phases to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) creatingCOSRWithRevisionZeroShouldFail() error {
	tc.resetCOSRBuilder("rev-zero", 0)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-rev-zero", tc.namespace))
	err := tc.createCOSR(context.Background())
	if err == nil {
		return fmt.Errorf("expected COSR with revision 0 to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) creatingCOSRWithUnsetLifecycleStateShouldFail() error {
	tc.resetCOSRBuilder("lcs-unset", 1)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-lcs-unset", tc.namespace))
	cosr := tc.buildCOSR()
	cosr.Spec.LifecycleState = ""
	if err := tc.client.Create(context.Background(), cosr); err == nil {
		return fmt.Errorf("expected COSR with unset lifecycleState to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) creatingCOSRWithUnknownLifecycleStateShouldFail() error {
	tc.resetCOSRBuilder("lcs-unknown", 1)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-lcs-unknown", tc.namespace))
	cosr := tc.buildCOSR()
	cosr.Spec.LifecycleState = "Unknown"
	if err := tc.client.Create(context.Background(), cosr); err == nil {
		return fmt.Errorf("expected COSR with unknown lifecycleState to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) cosrGroupOfLength(n int) string {
	prefix := tc.namespace + "-"
	pad := n - len(prefix)
	if pad < 0 {
		pad = 0
	}
	return prefix + strings.Repeat("a", pad)
}

func (tc *testContext) creatingCOSRWithExact52CharGroupShouldSucceed() error {
	tc.resetCOSRBuilder("x", 1)
	tc.cosr.group = tc.cosrGroupOfLength(52)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-exact-group", tc.namespace))
	return tc.createCOSR(context.Background())
}

func (tc *testContext) creatingCOSRWithLongGroupShouldFail() error {
	tc.resetCOSRBuilder("x", 1)
	tc.cosr.group = tc.cosrGroupOfLength(53)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-long-group", tc.namespace))
	err := tc.createCOSR(context.Background())
	if err == nil {
		return fmt.Errorf("expected COSR with group longer than 52 characters to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) updatingCOSRCollisionProtectionShouldFail() error {
	name := tc.lastCreatedCOSRName()
	cosr := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, cosr); err != nil {
		return err
	}
	cp := orbv1alpha1.CollisionProtectionNone
	cosr.Spec.CollisionProtection = &cp
	err := tc.client.Update(context.Background(), cosr)
	if err == nil {
		return fmt.Errorf("expected updating collisionProtection to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) allPhasesComplete() error {
	return nil
}

func (tc *testContext) theConfigMapFieldIsSetTo(name, field, value string) error {
	ctx := context.Background()
	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: tc.namespace, Name: name}
	if err := tc.client.Get(ctx, key, cm); err != nil {
		return fmt.Errorf("getting ConfigMap %q: %w", name, err)
	}
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	dataKey := strings.TrimPrefix(field, ".data.")
	cm.Data[dataKey] = value
	return tc.client.Update(ctx, cm)
}

func (tc *testContext) theConfigMapIsRecreatedByController(_ string) error {
	return nil
}

func (tc *testContext) theCOSRInGroupLifecycleStateIsSetTo(group string, revision uint32, state string) error {
	ctx := context.Background()
	name := tc.cosrName(group, revision)
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		cosr := &orbv1alpha1.ClusterObjectSetRevision{}
		if err := tc.client.Get(ctx, types.NamespacedName{Name: name}, cosr); err != nil {
			return false, err
		}
		cosr.Spec.LifecycleState = orbv1alpha1.LifecycleState(state)
		if err := tc.client.Update(ctx, cosr); err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (tc *testContext) aNewCOSRIsCreated(group string, revision uint32) {
	tc.resetCOSRBuilder(group, revision)
}

func (tc *testContext) theNewCOSRIsCreated() error {
	return tc.createCOSR(context.Background())
}

func (tc *testContext) theNewCOSRIsCreatedAndBecomesAvailable() error {
	if err := tc.createCOSR(context.Background()); err != nil {
		return err
	}
	return tc.pollForCOSRCondition(context.Background(), tc.lastCreatedCOSRName(), "Available", metav1.ConditionTrue)
}

func (tc *testContext) revisionIsArchived(revision uint32) error {
	for name, cosr := range tc.cosrs {
		if cosr.Spec.Revision == revision {
			latest := &orbv1alpha1.ClusterObjectSetRevision{}
			if err := tc.client.Get(context.Background(), types.NamespacedName{Name: name}, latest); err != nil {
				return err
			}
			latest.Spec.LifecycleState = orbv1alpha1.LifecycleStateArchived
			return tc.client.Update(context.Background(), latest)
		}
	}
	return fmt.Errorf("revision %d not found", revision)
}

func (tc *testContext) theCOSIsCreated() error {
	return tc.createCOS(context.Background())
}

func (tc *testContext) theCOSTemplateSpecIsUpdated(cmName, phaseName string) error {
	ctx := context.Background()
	name := tc.lastCreatedCOSName()
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		if err := tc.client.Get(ctx, types.NamespacedName{Name: name}, cos); err != nil {
			return false, err
		}
		cos.Spec.Template.Spec = orbv1alpha1.ClusterObjectSetTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name: phaseName,
				Objects: []orbv1alpha1.PhaseObject{{
					Object: runtime.RawExtension{Object: newConfigMap(cmName, tc.namespace)},
				}},
			}},
		}
		if err := tc.client.Update(ctx, cos); err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (tc *testContext) theCOSTemplateSpecIsUpdatedWithGatedConfigMap(cmName, phaseName string) error {
	ctx := context.Background()
	name := tc.lastCreatedCOSName()
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		if err := tc.client.Get(ctx, types.NamespacedName{Name: name}, cos); err != nil {
			return false, err
		}
		cos.Spec.Template.Spec = orbv1alpha1.ClusterObjectSetTemplateSpec{
			Phases: []orbv1alpha1.Phase{{
				Name: phaseName,
				Objects: []orbv1alpha1.PhaseObject{{
					Object: runtime.RawExtension{Object: newConfigMap(cmName, tc.namespace)},
					Assertions: []orbv1alpha1.Assertion{{
						FieldValue: &orbv1alpha1.FieldValueAssertion{
							FieldPath: ".data.ready",
							Value:     "true",
						},
					}},
				}},
			}},
		}
		if err := tc.client.Update(ctx, cos); err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (tc *testContext) theCOSTemplateLabelIsUpdated(key, value string) error {
	ctx := context.Background()
	name := tc.lastCreatedCOSName()
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		if err := tc.client.Get(ctx, types.NamespacedName{Name: name}, cos); err != nil {
			return false, err
		}
		if cos.Spec.Template.Metadata.Labels == nil {
			cos.Spec.Template.Metadata.Labels = make(map[string]string)
		}
		cos.Spec.Template.Metadata.Labels[key] = value
		if err := tc.client.Update(ctx, cos); err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (tc *testContext) theCOSLabelIsSetTo(cosName, key, value string) error {
	ctx := context.Background()
	fullName := tc.namespace + "-" + cosName
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		if err := tc.client.Get(ctx, types.NamespacedName{Name: fullName}, cos); err != nil {
			return false, err
		}
		if cos.Labels == nil {
			cos.Labels = make(map[string]string)
		}
		cos.Labels[key] = value
		if err := tc.client.Update(ctx, cos); err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (tc *testContext) theCOSRevisionHistoryLimitIsSetTo(cosName string, limit int32) error {
	ctx := context.Background()
	fullName := tc.namespace + "-" + cosName
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		if err := tc.client.Get(ctx, types.NamespacedName{Name: fullName}, cos); err != nil {
			return false, err
		}
		cos.Spec.RevisionHistoryLimit = &limit
		if err := tc.client.Update(ctx, cos); err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (tc *testContext) cosNameOfLength(n int) string {
	prefix := tc.namespace + "-"
	pad := n - len(prefix)
	if pad < 0 {
		pad = 0
	}
	return prefix + strings.Repeat("b", pad)
}

func (tc *testContext) creatingCOSWithExact52CharNameShouldSucceed() error {
	tc.resetCOSBuilder("x")
	tc.cos.name = tc.cosNameOfLength(52)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-exact-name", tc.namespace))
	return tc.createCOS(context.Background())
}

func (tc *testContext) creatingCOSWithLongNameShouldFail() error {
	tc.resetCOSBuilder("x")
	tc.cos.name = tc.cosNameOfLength(53)
	tc.addPhase("install")
	tc.addObjectToPhase(newConfigMap("cm-long-name", tc.namespace))
	err := tc.createCOS(context.Background())
	if err == nil {
		return fmt.Errorf("expected COS with name longer than 52 characters to fail, but it succeeded")
	}
	return nil
}

func (tc *testContext) theCOSIsDeleted(name string) error {
	cosName := tc.namespace + "-" + name
	cos := &orbv1alpha1.ClusterObjectSet{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: cosName}, cos); err != nil {
		return err
	}
	return tc.client.Delete(context.Background(), cos)
}

func (tc *testContext) theCOSIsDeletedWithCascadeOrphan(name string) error {
	cosName := tc.namespace + "-" + name
	cos := &orbv1alpha1.ClusterObjectSet{}
	if err := tc.client.Get(context.Background(), types.NamespacedName{Name: cosName}, cos); err != nil {
		return err
	}
	orphan := metav1.DeletePropagationOrphan
	return tc.client.Delete(context.Background(), cos, &client.DeleteOptions{
		PropagationPolicy: &orphan,
	})
}
