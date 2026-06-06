package e2e

import (
	"context"

	"github.com/cucumber/godog"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func registerSetupSteps(sc *godog.ScenarioContext, tc *testContext) {
	sc.Step(`^a COSR named "([^"]*)" with group "([^"]*)" and revision (\d+)$`, tc.aCOSRNamedWithGroupAndRevision)
	sc.Step(`^a COSR with group "([^"]*)" and revision (\d+)$`, tc.aCOSRWithGroupAndRevision)
	sc.Step(`^(?:a|the) phase "([^"]*)" (?:with|has) a ConfigMap "([^"]*)"$`, tc.aPhaseWithConfigMap)
	sc.Step(`^(?:a|the) phase "([^"]*)" (?:with|has) a gated ConfigMap "([^"]*)"$`, tc.aPhaseWithGatedConfigMap)
	sc.Step(`^the phase "([^"]*)" also has a ConfigMap "([^"]*)"$`, tc.phaseAlsoHasConfigMap)
	sc.Step(`^a phase "([^"]*)" with a CRD "([^"]*)"$`, tc.aPhaseWithCRD)
	sc.Step(`^the phase "([^"]*)" also has a CRD "([^"]*)"$`, tc.phaseAlsoHasCRD)
	sc.Step(`^a phase "([^"]*)" with a CRD "([^"]*)" with assertion conditionEqual type "([^"]*)" status "([^"]*)"$`, tc.aPhaseWithCRDConditionEqual)
	sc.Step(`^a phase "([^"]*)" with a "([^"]*)" named "([^"]*)"$`, tc.aPhaseWithCR)
	sc.Step(`^the phase "([^"]*)" also has a "([^"]*)" named "([^"]*)"$`, tc.phaseAlsoHasCR)
	sc.Step(`^a phase "([^"]*)" with a ConfigMap "([^"]*)" with assertion fieldValue path "([^"]*)" value "([^"]*)"$`, tc.aPhaseWithConfigMapFieldValue)
	sc.Step(`^a phase "([^"]*)" with a ConfigMap "([^"]*)" with assertion celExpression "([^"]*)"$`, tc.aPhaseWithConfigMapCEL)
	sc.Step(`^a phase "([^"]*)" with a ConfigMap "([^"]*)" with assertion celExpression "([^"]*)" message "([^"]*)"$`, tc.aPhaseWithConfigMapCELMessage)
	sc.Step(`^(?:a|the) phase "([^"]*)" (?:with|has) a ConfigMap "([^"]*)" with data:$`, tc.aPhaseWithConfigMapDataTable)
	sc.Step(`^a phase "([^"]*)" with a ConfigMap "([^"]*)" with data key "([^"]*)" value "([^"]*)"$`, tc.aPhaseWithConfigMapData)
	sc.Step(`^the phase "([^"]*)" also has a ConfigMap "([^"]*)" with data:$`, tc.phaseAlsoHasConfigMapDataTable)
	sc.Step(`^the last object has assertion conditionEqual type "([^"]*)" status "([^"]*)"$`, tc.lastObjectHasConditionEqualAssertion)
	sc.Step(`^the last object has assertion fieldsEqual fieldA "([^"]*)" fieldB "([^"]*)"$`, tc.lastObjectHasFieldsEqualAssertion)
	sc.Step(`^the last object has assertion fieldValue path "([^"]*)" value "([^"]*)"$`, tc.lastObjectHasFieldValueAssertion)
	sc.Step(`^the last object has assertion celExpression "([^"]*)"$`, tc.lastObjectHasCELAssertion)
	sc.Step(`^the COSR collisionProtection is "([^"]*)"$`, tc.theCOSRCollisionProtectionIs)
	sc.Step(`^the phase "([^"]*)" collisionProtection is "([^"]*)"$`, tc.thePhaseCollisionProtectionIs)
	sc.Step(`^the last object collisionProtection is "([^"]*)"$`, tc.theLastObjectCollisionProtectionIs)
	sc.Step(`^a standalone ConfigMap "([^"]*)" exists$`, tc.aStandaloneConfigMapExists)

	sc.Step(`^a phase "([^"]*)" with an unregistered resource type$`, tc.aPhaseWithUnregisteredResourceType)

	sc.Step(`^an available COSR with group "([^"]*)" and revision (\d+)$`, tc.anAvailableCOSR)

	// COS setup steps
	sc.Step(`^a COS named "([^"]*)"$`, tc.aCOSNamed)
	sc.Step(`^a COS named "([^"]*)" with revisionHistoryLimit (\d+)$`, tc.aCOSNamedWithRevisionHistoryLimit)
	sc.Step(`^an available COS named "([^"]*)"$`, tc.anAvailableCOS)
	sc.Step(`^the COS template has (label|annotation) "([^"]*)" with value "([^"]*)"$`, tc.theCOSTemplateHasMetadata)
}

func (tc *testContext) aCOSRNamedWithGroupAndRevision(name, group string, revision uint32) {
	tc.resetCOSRBuilder(group, revision)
	tc.cosr.nameOverride = name
}

func (tc *testContext) anAvailableCOSR(group string, revision uint32) error {
	tc.resetCOSRBuilder(group, revision)
	tc.addPhase("install")
	tc.addConfigMapToPhase("cm-"+group, false)
	if err := tc.createCOSR(context.Background()); err != nil {
		return err
	}
	return tc.pollForCOSRCondition(context.Background(), tc.lastCreatedCOSRName(), "Available", metav1.ConditionTrue)
}

func (tc *testContext) aCOSRWithGroupAndRevision(group string, revision uint32) {
	tc.resetCOSRBuilder(group, revision)
}

func (tc *testContext) addConfigMapToPhase(name string, gated bool) {
	assertion := openByDefaultGateAssertion
	if gated {
		assertion = closedByDefaultGateAssertion
	}
	phase := tc.currentPhase()
	phase.Objects = append(phase.Objects, orbv1alpha1.PhaseObject{
		Object:     runtime.RawExtension{Object: newConfigMap(name, tc.namespace)},
		Assertions: []orbv1alpha1.Assertion{assertion},
	})
}

func (tc *testContext) aPhaseWithConfigMap(phaseName, cmName string) {
	tc.addPhase(phaseName)
	tc.addConfigMapToPhase(cmName, false)
}

func (tc *testContext) aPhaseWithGatedConfigMap(phaseName, cmName string) {
	tc.addPhase(phaseName)
	tc.addConfigMapToPhase(cmName, true)
}

func (tc *testContext) phaseAlsoHasConfigMap(_, cmName string) {
	tc.addConfigMapToPhase(cmName, false)
}

func (tc *testContext) aPhaseWithCRDConditionEqual(phaseName, crdName, condType, condStatus string) {
	tc.addPhase(phaseName)
	crd := newCRD(crdName)
	tc.crds = append(tc.crds, crd.Name)
	tc.addObjectWithAssertions(crd, []orbv1alpha1.Assertion{{
		ConditionEqual: &orbv1alpha1.ConditionEqualAssertion{
			Type:   condType,
			Status: condStatus,
		},
	}})
}

func (tc *testContext) aPhaseWithCRD(phaseName, crdName string) {
	tc.addPhase(phaseName)
	crd := newCRD(crdName)
	tc.crds = append(tc.crds, crd.Name)
	tc.addObjectToPhase(crd)
}

func (tc *testContext) phaseAlsoHasCRD(_, crdName string) {
	crd := newCRD(crdName)
	tc.crds = append(tc.crds, crd.Name)
	tc.addObjectToPhase(crd)
}

func (tc *testContext) aPhaseWithCR(phaseName, crdName, crName string) {
	tc.addPhase(phaseName)
	tc.addObjectToPhase(newCR(crdName, crName))
}

func (tc *testContext) phaseAlsoHasCR(_, crdName, crName string) {
	tc.addObjectToPhase(newCR(crdName, crName))
}

func (tc *testContext) aPhaseWithConfigMapFieldValue(phaseName, cmName, path, value string) {
	tc.addPhase(phaseName)
	tc.addObjectWithAssertions(newConfigMap(cmName, tc.namespace), []orbv1alpha1.Assertion{{
		FieldValue: &orbv1alpha1.FieldValueAssertion{
			FieldPath: path,
			Value:     value,
		},
	}})
}

func (tc *testContext) aPhaseWithConfigMapCEL(phaseName, cmName, expr string) {
	tc.addPhase(phaseName)
	tc.addObjectWithAssertions(newConfigMap(cmName, tc.namespace), []orbv1alpha1.Assertion{{
		CELExpression: &orbv1alpha1.CELExpressionAssertion{
			Expression: expr,
		},
	}})
}

func (tc *testContext) aPhaseWithConfigMapCELMessage(phaseName, cmName, expr, message string) {
	tc.addPhase(phaseName)
	tc.addObjectWithAssertions(newConfigMap(cmName, tc.namespace), []orbv1alpha1.Assertion{{
		CELExpression: &orbv1alpha1.CELExpressionAssertion{
			Expression: expr,
			Message:    message,
		},
	}})
}

func (tc *testContext) aPhaseWithConfigMapDataTable(phaseName, cmName string, table *godog.Table) {
	tc.addPhase(phaseName)
	tc.addObjectToPhase(newConfigMapWithData(cmName, tc.namespace, tableToMap(table)))
}

func (tc *testContext) aPhaseWithConfigMapData(phaseName, cmName, key, value string) {
	tc.addPhase(phaseName)
	tc.addObjectToPhase(newConfigMapWithData(cmName, tc.namespace, map[string]string{key: value}))
}

func (tc *testContext) phaseAlsoHasConfigMapDataTable(_, cmName string, table *godog.Table) {
	tc.addObjectToPhase(newConfigMapWithData(cmName, tc.namespace, tableToMap(table)))
}

func (tc *testContext) lastObjectHasConditionEqualAssertion(condType, condStatus string) {
	obj := tc.lastObject()
	obj.Assertions = append(obj.Assertions, orbv1alpha1.Assertion{
		ConditionEqual: &orbv1alpha1.ConditionEqualAssertion{
			Type:   condType,
			Status: condStatus,
		},
	})
}

func (tc *testContext) lastObjectHasFieldsEqualAssertion(fieldA, fieldB string) {
	obj := tc.lastObject()
	obj.Assertions = append(obj.Assertions, orbv1alpha1.Assertion{
		FieldsEqual: &orbv1alpha1.FieldsEqualAssertion{
			FieldA: fieldA,
			FieldB: fieldB,
		},
	})
}

func (tc *testContext) lastObjectHasFieldValueAssertion(path, value string) {
	obj := tc.lastObject()
	obj.Assertions = append(obj.Assertions, orbv1alpha1.Assertion{
		FieldValue: &orbv1alpha1.FieldValueAssertion{
			FieldPath: path,
			Value:     value,
		},
	})
}

func (tc *testContext) lastObjectHasCELAssertion(expr string) {
	obj := tc.lastObject()
	obj.Assertions = append(obj.Assertions, orbv1alpha1.Assertion{
		CELExpression: &orbv1alpha1.CELExpressionAssertion{
			Expression: expr,
		},
	})
}

func (tc *testContext) theCOSRCollisionProtectionIs(cp string) {
	v := orbv1alpha1.CollisionProtection(cp)
	tc.tmpl.collisionProtection = &v
}

func (tc *testContext) thePhaseCollisionProtectionIs(phaseName, cp string) {
	v := orbv1alpha1.CollisionProtection(cp)
	for i := range tc.tmpl.phases {
		if tc.tmpl.phases[i].Name == phaseName {
			tc.tmpl.phases[i].CollisionProtection = &v
			return
		}
	}
}

func (tc *testContext) theLastObjectCollisionProtectionIs(cp string) {
	v := orbv1alpha1.CollisionProtection(cp)
	tc.lastObject().CollisionProtection = &v
}

func (tc *testContext) aStandaloneConfigMapExists(name string) error {
	return tc.client.Create(context.Background(), newConfigMap(name, tc.namespace))
}

func (tc *testContext) aCOSNamed(name string) {
	tc.resetCOSBuilder(name)
}

func (tc *testContext) anAvailableCOS(name string) error {
	tc.resetCOSBuilder(name)
	tc.addPhase("install")
	tc.addConfigMapToPhase("cm-"+name, false)
	if err := tc.createCOS(context.Background()); err != nil {
		return err
	}
	return tc.theCOSShouldBeAvailable(name)
}

func (tc *testContext) aCOSNamedWithRevisionHistoryLimit(name string, limit int32) {
	tc.resetCOSBuilder(name)
	tc.cos.revisionHistoryLimit = &limit
}

func (tc *testContext) theCOSTemplateHasMetadata(kind, key, value string) {
	if kind == "label" {
		if tc.cos.labels == nil {
			tc.cos.labels = make(map[string]string)
		}
		tc.cos.labels[key] = value
	} else {
		if tc.cos.annotations == nil {
			tc.cos.annotations = make(map[string]string)
		}
		tc.cos.annotations[key] = value
	}
}

func newConfigMap(name, namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newConfigMapWithData(name, namespace string, data map[string]string) *corev1.ConfigMap {
	cm := newConfigMap(name, namespace)
	cm.Data = data
	return cm
}

var openByDefaultGateAssertion = orbv1alpha1.Assertion{
	CELExpression: &orbv1alpha1.CELExpressionAssertion{
		Expression: "!has(self.data) || !has(self.data.gate) || self.data.gate != 'closed'",
	},
}

var closedByDefaultGateAssertion = orbv1alpha1.Assertion{
	CELExpression: &orbv1alpha1.CELExpressionAssertion{
		Expression: "has(self.data) && has(self.data.gate) && self.data.gate == 'open'",
	},
}

func newGatedConfigMapPhaseObject(name, namespace string, gated bool) orbv1alpha1.PhaseObject {
	assertion := openByDefaultGateAssertion
	if gated {
		assertion = closedByDefaultGateAssertion
	}
	return orbv1alpha1.PhaseObject{
		Object:     runtime.RawExtension{Object: newConfigMap(name, namespace)},
		Assertions: []orbv1alpha1.Assertion{assertion},
	}
}

func newCRD(name string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name + ".e2e.orb.dev",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "e2e.orb.dev",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   name,
				Singular: name[:len(name)-1],
				Kind:     capitalize(name[:len(name)-1]),
			},
			Scope: apiextensionsv1.ClusterScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:    "v1alpha1",
				Served:  true,
				Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type:                   "object",
						XPreserveUnknownFields: boolPtr(true),
					},
				},
			}},
		},
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func boolPtr(b bool) *bool {
	return &b
}

func newCR(crdName, crName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "e2e.orb.dev/v1alpha1",
			"kind":       capitalize(crdName[:len(crdName)-1]),
			"metadata": map[string]interface{}{
				"name": crName,
			},
		},
	}
}

func (tc *testContext) aPhaseWithUnregisteredResourceType(phaseName string) {
	tc.addPhase(phaseName)
	tc.addObjectToPhase(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "totally.fake.example.com/v1",
			"kind":       "Nonexistent",
			"metadata": map[string]interface{}{
				"name": "fake-resource",
			},
		},
	})
}

func tableToMap(table *godog.Table) map[string]string {
	data := make(map[string]string)
	for _, row := range table.Rows[1:] {
		data[row.Cells[0].Value] = row.Cells[1].Value
	}
	return data
}
