/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package securitygroups

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/golang/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	infrav1 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha2"
	"sigs.k8s.io/cluster-api-provider-azure/cloud/scope"
	"sigs.k8s.io/cluster-api-provider-azure/cloud/services/securitygroups/mock_securitygroups"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileSecurityGroups(t *testing.T) {
	testcases := []struct {
		name           string
		sgName         string
		isControlPlane bool
		vnetSpec       *infrav1.VnetSpec
		expect         func(m *mock_securitygroups.MockClientMockRecorder)
	}{
		{
			name:           "security group does not exists",
			sgName:         "my-sg",
			isControlPlane: true,
			vnetSpec:       &infrav1.VnetSpec{},
			expect: func(m *mock_securitygroups.MockClientMockRecorder) {
				m.CreateOrUpdate(context.TODO(), "my-rg", "my-sg", gomock.AssignableToTypeOf(network.SecurityGroup{}))
			},
		}, {
			name:           "security group does not exist and it's not for a control plane",
			sgName:         "my-sg",
			isControlPlane: false,
			vnetSpec:       &infrav1.VnetSpec{},
			expect: func(m *mock_securitygroups.MockClientMockRecorder) {
				m.CreateOrUpdate(context.TODO(), "my-rg", "my-sg", gomock.AssignableToTypeOf(network.SecurityGroup{}))
			},
		}, {
			name:           "skipping network security group reconcile in custom vnet mode",
			sgName:         "my-sg",
			isControlPlane: false,
			vnetSpec:       &infrav1.VnetSpec{ResourceGroup: "custom-vnet-rg", Name: "custom-vnet", ID: "id1"},
			expect: func(m *mock_securitygroups.MockClientMockRecorder) {

			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			sgMock := mock_securitygroups.NewMockClient(mockCtrl)

			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			}

			client := fake.NewFakeClient(cluster)

			tc.expect(sgMock.EXPECT())

			clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
				AzureClients: scope.AzureClients{
					SubscriptionID: "123",
					Authorizer:     autorest.NullAuthorizer{},
				},
				Client:  client,
				Cluster: cluster,
				AzureCluster: &infrav1.AzureCluster{
					Spec: infrav1.AzureClusterSpec{
						Location: "test-location",
						ResourceGroup: "my-rg",
						NetworkSpec: infrav1.NetworkSpec{
							Vnet: *tc.vnetSpec,
						},
					},
				},
			})
			if err != nil {
				t.Fatalf("Failed to create test context: %v", err)
			}

			s := &Service{
				Scope:  clusterScope,
				Client: sgMock,
			}

			sgSpec := &Spec{
				Name:           tc.sgName,
				IsControlPlane: tc.isControlPlane,
			}
			if err := s.Reconcile(context.TODO(), sgSpec); err != nil {
				t.Fatalf("got an unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteSecurityGroups(t *testing.T) {
	testcases := []struct {
		name   string
		sgName string
		expect func(m *mock_securitygroups.MockClientMockRecorder)
	}{
		{
			name:   "security group exists",
			sgName: "my-sg",
			expect: func(m *mock_securitygroups.MockClientMockRecorder) {
				m.Delete(context.TODO(), "my-rg", "my-sg")
			},
		},
		{
			name:   "security group already deleted",
			sgName: "my-sg",
			expect: func(m *mock_securitygroups.MockClientMockRecorder) {
				m.Delete(context.TODO(), "my-rg", "my-sg").
					Return(autorest.NewErrorWithResponse("", "", &http.Response{StatusCode: 404}, "Not found"))
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			sgMock := mock_securitygroups.NewMockClient(mockCtrl)

			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			}

			client := fake.NewFakeClient(cluster)

			tc.expect(sgMock.EXPECT())

			clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
				AzureClients: scope.AzureClients{
					SubscriptionID: "123",
					Authorizer:     autorest.NullAuthorizer{},
				},
				Client:  client,
				Cluster: cluster,
				AzureCluster: &infrav1.AzureCluster{
					Spec: infrav1.AzureClusterSpec{
						Location: "test-location",
						ResourceGroup: "my-rg",
					},
				},
			})
			if err != nil {
				t.Fatalf("Failed to create test context: %v", err)
			}

			s := &Service{
				Scope:  clusterScope,
				Client: sgMock,
			}

			sgSpec := &Spec{
				Name:           tc.sgName,
				IsControlPlane: false,
			}

			if err := s.Delete(context.TODO(), sgSpec); err != nil {
				t.Fatalf("got an unexpected error: %v", err)
			}
		})
	}
}
