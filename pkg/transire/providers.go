// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package transire

// ProviderFactory is a function type that creates providers
type ProviderFactory func(region string) Provider

// Global provider registry - populated by provider packages via init()
var providerFactories = map[string]ProviderFactory{}

// RegisterProviderFactory registers a provider factory for a given cloud
func RegisterProviderFactory(cloud string, factory ProviderFactory) {
	providerFactories[cloud] = factory
}
