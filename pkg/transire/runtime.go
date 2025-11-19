// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package transire

// RuntimeType represents the execution environment (for logging/debugging only)
// NOTE: Runtime detection REMOVED. Build tags select runtime at compile time.
type RuntimeType string

const (
	RuntimeLocal     RuntimeType = "local"
	RuntimeAWSLambda RuntimeType = "aws_lambda"
	RuntimeGCPRun    RuntimeType = "gcp_cloudrun"   // Future
	RuntimeAzureFunc RuntimeType = "azure_function" // Future
)
