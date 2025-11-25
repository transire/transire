// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dispatcher

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/transire/transire"
	"github.com/transire/transire/dispatcher/aws"
	"github.com/transire/transire/dispatcher/local"
)

const dispatcherEnv = "TRANSIRE_DISPATCHER"

// ErrUnknownDispatcher indicates an invalid dispatcher override value.
var ErrUnknownDispatcher = errors.New("transire: unknown dispatcher override")

// Auto selects the best dispatcher for the current environment.
// Priority:
//  1. TRANSIRE_DISPATCHER=aws|local (explicit override)
//  2. Lambda environment variables present => AWS dispatcher
//  3. Default to local dispatcher
func Auto() (transire.Dispatcher, error) {
	if override := strings.TrimSpace(os.Getenv(dispatcherEnv)); override != "" {
		switch strings.ToLower(override) {
		case "aws", "lambda":
			return &aws.Dispatcher{}, nil
		case "local":
			return &local.Dispatcher{}, nil
		default:
			return nil, fmt.Errorf("%w: %s", ErrUnknownDispatcher, override)
		}
	}

	if isLambdaEnv() {
		return &aws.Dispatcher{}, nil
	}

	return &local.Dispatcher{}, nil
}

func isLambdaEnv() bool {
	return os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" ||
		os.Getenv("AWS_EXECUTION_ENV") != "" ||
		os.Getenv("LAMBDA_TASK_ROOT") != ""
}
