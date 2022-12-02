/*
Copyright 2022.

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

package internalrequest

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/redhat-appstudio/internal-services/api/v1alpha1"
	"github.com/redhat-appstudio/operator-goodies/reconciler"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Adapter holds the objects needed to reconcile a Release.
type Adapter struct {
	client          client.Client
	context         context.Context
	internalClient  client.Client
	internalRequest *v1alpha1.InternalRequest
	logger          logr.Logger
}

// NewAdapter creates and returns an Adapter instance.
func NewAdapter(internalRequest *v1alpha1.InternalRequest, client, internalClient client.Client, context context.Context, logger logr.Logger) *Adapter {
	return &Adapter{
		client:          client,
		context:         context,
		internalRequest: internalRequest,
		internalClient:  internalClient,
		logger:          logger,
	}
}

func (a *Adapter) EnsureReconcileOperationIsLogged() (reconciler.OperationResult, error) {
	a.logger.Info("InternalRequest successfully watched")

	return reconciler.ContinueProcessing()
}
