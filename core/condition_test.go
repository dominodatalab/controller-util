package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockConditionObject adds conditions to a real Kubernetes object
// Use a concrete Kubernetes type (Deployment) that implements
// client.Object instead of implementing client.Object in the test
type MockConditionObject struct {
	*appsv1.Deployment
	Conditions []metav1.Condition
}

func (m *MockConditionObject) GetConditions() *[]metav1.Condition {
	return &m.Conditions
}

func NewMockObject(name string, generation int64) *MockConditionObject {
	return &MockConditionObject{
		Deployment: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:       name,
				Generation: generation,
			},
		},
		Conditions: []metav1.Condition{},
	}
}

func TestConditionHelper_New(t *testing.T) {
	obj := NewMockObject("test-object", 42)

	helper := NewConditionHelper(obj)
	assert.NotNil(t, helper)
	assert.Equal(t, client.Object(obj), helper.obj)
	assert.Empty(t, helper.pending)
}

func TestConditionHelper_SetCondition(t *testing.T) {
	obj := NewMockObject("test-object", 42)
	helper := NewConditionHelper(obj)

	cond := metav1.Condition{
		Type:   "TestCondition",
		Status: metav1.ConditionTrue,
		Reason: "Testing",
	}

	helper.SetCondition(cond)

	// Verify it's in pending but not in object yet
	assert.Len(t, helper.pending, 1)
	assert.Contains(t, helper.pending, "TestCondition")
	assert.Empty(t, obj.Conditions)

	// Verify generation was set if not provided
	assert.Equal(t, int64(42), helper.pending["TestCondition"].ObservedGeneration)
}

func TestConditionHelper_Set(t *testing.T) {
	obj := NewMockObject("test-object", 42)
	helper := NewConditionHelper(obj)

	helper.Set("TestCondition", metav1.ConditionTrue, "Testing", "This is a test")

	assert.Len(t, helper.pending, 1)
	assert.Contains(t, helper.pending, "TestCondition")

	pendingCond := helper.pending["TestCondition"]
	assert.Equal(t, "TestCondition", pendingCond.Type)
	assert.Equal(t, metav1.ConditionTrue, pendingCond.Status)
	assert.Equal(t, "Testing", pendingCond.Reason)
	assert.Equal(t, "This is a test", pendingCond.Message)
	assert.Equal(t, int64(42), pendingCond.ObservedGeneration)
}

func TestConditionHelper_Setf(t *testing.T) {
	obj := NewMockObject("test-object", 42)
	helper := NewConditionHelper(obj)

	helper.Setf("TestCondition", metav1.ConditionTrue, "Testing", "Count: %d", 5)

	assert.Equal(t, "Count: 5", helper.pending["TestCondition"].Message)
}

func TestConditionHelper_SetTrue(t *testing.T) {
	obj := NewMockObject("test-object", 1)
	helper := NewConditionHelper(obj)

	helper.SetTrue("TestCondition", "Success", "Operation completed")

	assert.Equal(t, metav1.ConditionTrue, helper.pending["TestCondition"].Status)
}

func TestConditionHelper_SetFalse(t *testing.T) {
	obj := NewMockObject("test-object", 1)
	helper := NewConditionHelper(obj)

	helper.SetFalse("TestCondition", "Failure", "Operation failed")

	assert.Equal(t, metav1.ConditionFalse, helper.pending["TestCondition"].Status)
}

func TestConditionHelper_SetUnknown(t *testing.T) {
	obj := NewMockObject("test-object", 1)
	helper := NewConditionHelper(obj)

	helper.SetUnknown("TestCondition", "Unknown", "Status is unknown")

	assert.Equal(t, metav1.ConditionUnknown, helper.pending["TestCondition"].Status)
}

func TestConditionHelper_SetfUnknown(t *testing.T) {
	obj := NewMockObject("test-object", 1)
	helper := NewConditionHelper(obj)

	helper.SetfUnknown("TestCondition", "Unknown", "Status for %s is unknown", "resource")

	assert.Equal(t, metav1.ConditionUnknown, helper.pending["TestCondition"].Status)
	assert.Equal(t, "Status for resource is unknown", helper.pending["TestCondition"].Message)
}

func TestConditionHelper_Flush(t *testing.T) {
	obj := NewMockObject("test-object", 1)
	helper := NewConditionHelper(obj)

	helper.SetTrue("Condition1", "Reason1", "Message1")
	helper.SetFalse("Condition2", "Reason2", "Message2")

	err := helper.Flush()

	assert.NoError(t, err)
	assert.Len(t, obj.Conditions, 2)

	var cond1Index, cond2Index int
	var found1, found2 bool

	cond1Index, found1 = FindStatusCondition(obj.Conditions, "Condition1")
	cond2Index, found2 = FindStatusCondition(obj.Conditions, "Condition2")

	assert.True(t, found1, "Should find Condition1")
	assert.True(t, found2, "Should find Condition2")

	assert.Equal(t, metav1.ConditionTrue, obj.Conditions[cond1Index].Status)
	assert.Equal(t, "Reason1", obj.Conditions[cond1Index].Reason)
	assert.Equal(t, "Message1", obj.Conditions[cond1Index].Message)

	assert.Equal(t, metav1.ConditionFalse, obj.Conditions[cond2Index].Status)
	assert.Equal(t, "Reason2", obj.Conditions[cond2Index].Reason)
	assert.Equal(t, "Message2", obj.Conditions[cond2Index].Message)

	assert.Empty(t, helper.pending)
}

func TestConditionHelper_FlushNonConditionObject(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name": "test-cm",
			},
		},
	}

	helper := NewConditionHelper(obj)

	helper.SetTrue("Condition", "Reason", "Message")

	err := helper.Flush()
	assert.NoError(t, err)
}

func TestSetStatusCondition(t *testing.T) {
	conditions := []metav1.Condition{}

	newCondition := metav1.Condition{
		Type:               "TestCondition",
		Status:             metav1.ConditionTrue,
		Reason:             "Testing",
		Message:            "This is a test",
		ObservedGeneration: 1,
	}

	SetStatusCondition(&conditions, newCondition)

	assert.Len(t, conditions, 1)
	assert.Equal(t, "TestCondition", conditions[0].Type)
	assert.Equal(t, metav1.ConditionTrue, conditions[0].Status)
	assert.NotZero(t, conditions[0].LastTransitionTime)

	// Modify the condition with status change
	updatedCondition := metav1.Condition{
		Type:               "TestCondition",
		Status:             metav1.ConditionFalse,
		Reason:             "UpdatedReason",
		Message:            "Updated message",
		ObservedGeneration: 2,
	}
	SetStatusCondition(&conditions, updatedCondition)

	idx, found := FindStatusCondition(conditions, "TestCondition")
	assert.True(t, found, "Condition should be found")

	assert.Equal(t, metav1.ConditionFalse, conditions[idx].Status, "Status should be updated to False")
	assert.Equal(t, "UpdatedReason", conditions[idx].Reason, "Reason should be updated")
	assert.Equal(t, "Updated message", conditions[idx].Message, "Message should be updated")
	assert.Equal(t, int64(2), conditions[idx].ObservedGeneration, "ObservedGeneration should be updated")
}

func TestSetStatusConditionWithCustomTime(t *testing.T) {
	pastTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	customTime := metav1.NewTime(pastTime)

	conditions := []metav1.Condition{}

	newCondition := metav1.Condition{
		Type:               "TestCondition",
		Status:             metav1.ConditionTrue,
		Reason:             "Testing",
		Message:            "This is a test",
		LastTransitionTime: customTime,
	}

	SetStatusCondition(&conditions, newCondition)

	assert.Equal(t, pastTime.Unix(), conditions[0].LastTransitionTime.Time.Unix())

	laterTime := time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)
	newCustomTime := metav1.NewTime(laterTime)

	updatedCondition := metav1.Condition{
		Type:               "TestCondition",
		Status:             metav1.ConditionFalse, // Changed status
		Reason:             "Updated",
		Message:            "Updated message",
		LastTransitionTime: newCustomTime,
	}

	SetStatusCondition(&conditions, updatedCondition)

	idx, found := FindStatusCondition(conditions, "TestCondition")
	assert.True(t, found, "Condition should be found")

	assert.Equal(t, laterTime.Unix(), conditions[idx].LastTransitionTime.Time.Unix())
}

func TestFindStatusCondition(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:    "Condition1",
			Status:  metav1.ConditionTrue,
			Reason:  "Reason1",
			Message: "Message1",
		},
		{
			Type:    "Condition2",
			Status:  metav1.ConditionFalse,
			Reason:  "Reason2",
			Message: "Message2",
		},
	}

	index, found := FindStatusCondition(conditions, "Condition2")

	assert.True(t, found, "Condition2 should be found")
	assert.Equal(t, 1, index, "Condition2 should be at index 1")
	assert.Equal(t, "Condition2", conditions[index].Type)
	assert.Equal(t, metav1.ConditionFalse, conditions[index].Status)

	index, found = FindStatusCondition(conditions, "NonExistentCondition")

	assert.False(t, found, "NonExistentCondition should not be found")
	assert.Equal(t, -1, index, "NonExistentCondition should return index -1")

	index, found = FindStatusCondition([]metav1.Condition{}, "AnyCondition")
	assert.False(t, found, "AnyCondition should not be found in empty slice")
	assert.Equal(t, -1, index, "Empty slice should return index -1")
}

func TestConditionFields(t *testing.T) {
	conditions := []metav1.Condition{}

	testCases := []struct {
		name     string
		condType string
		status   metav1.ConditionStatus
		reason   string
		message  string
	}{
		{
			name:     "True condition",
			condType: "Available",
			status:   metav1.ConditionTrue,
			reason:   "ResourceAvailable",
			message:  "Resource is available",
		},
		{
			name:     "False condition",
			condType: "Ready",
			status:   metav1.ConditionFalse,
			reason:   "ResourceNotReady",
			message:  "Resource is not ready",
		},
		{
			name:     "Unknown condition",
			condType: "Progressing",
			status:   metav1.ConditionUnknown,
			reason:   "Reconciling",
			message:  "Resource is being reconciled",
		},
		{
			name:     "With special characters",
			condType: "Validated",
			status:   metav1.ConditionTrue,
			reason:   "Test-Passed",
			message:  "Test passed with score: 99%",
		},
		{
			name:     "Empty message",
			condType: "Initialized",
			status:   metav1.ConditionTrue,
			reason:   "Completed",
			message:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newCondition := metav1.Condition{
				Type:    tc.condType,
				Status:  tc.status,
				Reason:  tc.reason,
				Message: tc.message,
			}

			SetStatusCondition(&conditions, newCondition)

			idx, found := FindStatusCondition(conditions, tc.condType)

			assert.True(t, found, "Condition should be found")
			assert.Equal(t, tc.condType, conditions[idx].Type)
			assert.Equal(t, tc.status, conditions[idx].Status)
			assert.Equal(t, tc.reason, conditions[idx].Reason)
			assert.Equal(t, tc.message, conditions[idx].Message)
		})
	}
}

func TestConditionTypeHandling(t *testing.T) {
	conditions := []metav1.Condition{}

	initialCond := metav1.Condition{
		Type:    "TestType",
		Status:  metav1.ConditionTrue,
		Reason:  "Initial",
		Message: "Initial condition",
	}
	SetStatusCondition(&conditions, initialCond)

	differentCaseCond := metav1.Condition{
		Type:    "testtype", // lowercase
		Status:  metav1.ConditionFalse,
		Reason:  "Different",
		Message: "Different case condition",
	}
	SetStatusCondition(&conditions, differentCaseCond)

	// This should create a new condition, not update the existing one
	assert.Len(t, conditions, 2)

	// Find both conditions
	idx1, found1 := FindStatusCondition(conditions, "TestType")
	idx2, found2 := FindStatusCondition(conditions, "testtype")

	assert.True(t, found1, "TestType should be found")
	assert.True(t, found2, "testtype should be found")
	assert.NotEqual(t, idx1, idx2, "The conditions should have different indices")

	// Test finding with wrong case
	idx, found := FindStatusCondition(conditions, "TesttYPE")
	assert.False(t, found, "TesttYPE should not be found (case sensitive)")
	assert.Equal(t, -1, idx, "Not found should return index -1")
}

func TestConditionStatusUpdates(t *testing.T) {
	conditions := []metav1.Condition{}
	initialCond := metav1.Condition{
		Type:    "TestStatus",
		Status:  metav1.ConditionTrue,
		Reason:  "Initial",
		Message: "Initial message",
	}
	SetStatusCondition(&conditions, initialCond)

	assert.Len(t, conditions, 1)
	assert.Equal(t, metav1.ConditionTrue, conditions[0].Status)

	originalTime := conditions[0].LastTransitionTime

	time.Sleep(10 * time.Millisecond)

	falseStatusCond := metav1.Condition{
		Type:    "TestStatus",
		Status:  metav1.ConditionFalse,
		Reason:  "Changed",
		Message: "Changed to false",
	}
	SetStatusCondition(&conditions, falseStatusCond)

	idx, found := FindStatusCondition(conditions, "TestStatus")
	assert.True(t, found, "TestStatus should be found")

	assert.Equal(t, metav1.ConditionFalse, conditions[idx].Status, "Status should change to False")
	assert.True(t, conditions[idx].LastTransitionTime.After(originalTime.Time),
		"LastTransitionTime should update when status changes from True to False")
}

func TestConditionReasonAndMessage(t *testing.T) {
	conditions := []metav1.Condition{}

	initialCond := metav1.Condition{
		Type:    "TestReason",
		Status:  metav1.ConditionTrue,
		Reason:  "InitialReason",
		Message: "Initial message",
	}
	SetStatusCondition(&conditions, initialCond)

	assert.Len(t, conditions, 1)
	assert.Equal(t, "InitialReason", conditions[0].Reason)
	assert.Equal(t, "Initial message", conditions[0].Message)

	newReasonCond := metav1.Condition{
		Type:    "TestReason",
		Status:  metav1.ConditionTrue, // Same status
		Reason:  "UpdatedReason",      // New reason
		Message: "Initial message",    // Same message
	}
	SetStatusCondition(&conditions, newReasonCond)

	idx, found := FindStatusCondition(conditions, "TestReason")
	assert.True(t, found, "TestReason should be found")

	// Verify reason was updated but message is unchanged
	assert.Equal(t, "UpdatedReason", conditions[idx].Reason, "Reason should be updated")
	assert.Equal(t, "Initial message", conditions[idx].Message, "Message should remain unchanged")
}

func TestConditionHelperSetFieldsExplicitly(t *testing.T) {
	testCases := []struct {
		name           string
		helperFunc     func(*conditionHelper, string, string, string)
		expectedStatus metav1.ConditionStatus
	}{
		{
			name: "SetTrue",
			helperFunc: func(h *conditionHelper, condType, reason, message string) {
				h.SetTrue(condType, reason, message)
			},
			expectedStatus: metav1.ConditionTrue,
		},
		{
			name: "SetFalse",
			helperFunc: func(h *conditionHelper, condType, reason, message string) {
				h.SetFalse(condType, reason, message)
			},
			expectedStatus: metav1.ConditionFalse,
		},
		{
			name: "SetUnknown",
			helperFunc: func(h *conditionHelper, condType, reason, message string) {
				h.SetUnknown(condType, reason, message)
			},
			expectedStatus: metav1.ConditionUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj := NewMockObject("test-object", 1)
			helper := NewConditionHelper(obj)

			tc.helperFunc(helper, "TestCondition", "TestReason", "Test message")

			assert.Len(t, helper.pending, 1)
			assert.Contains(t, helper.pending, "TestCondition")

			cond := helper.pending["TestCondition"]
			assert.Equal(t, "TestCondition", cond.Type)
			assert.Equal(t, tc.expectedStatus, cond.Status)
			assert.Equal(t, "TestReason", cond.Reason)
			assert.Equal(t, "Test message", cond.Message)
		})
	}
}
