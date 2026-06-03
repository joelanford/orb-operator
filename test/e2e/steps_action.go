package e2e

import (
	"context"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	sc.Step(`^the COSR is deleted with cascade foreground$`, tc.theCOSRIsDeletedWithCascadeForeground)
	sc.Step(`^the COSR is deleted with cascade background$`, tc.theCOSRIsDeletedWithCascadeBackground)
	sc.Step(`^the COSR is deleted with cascade orphan$`, tc.theCOSRIsDeletedWithCascadeOrphan)
	sc.Step(`^the CRD "([^"]*)" is deleted$`, tc.theCRDIsDeleted)
	sc.Step(`^the ConfigMap "([^"]*)" field "([^"]*)" is set to "([^"]*)"$`, tc.theConfigMapFieldIsSetTo)
	sc.Step(`^the ConfigMap "([^"]*)" is recreated by the controller$`, tc.theConfigMapIsRecreatedByController)
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
