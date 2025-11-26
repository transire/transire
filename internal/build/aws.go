// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package build

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/transire/transire/internal/config"
	"github.com/transire/transire/internal/discover"
)

const queueEnvPrefix = "TRANSIRE_QUEUE_"
const queueNameEnvSuffix = "_NAME"
const scheduleEnvPrefix = "TRANSIRE_SCHEDULE_"

// BuildAWS builds the Lambda bootstrap binary and generates CDK app files.
func BuildAWS(ctx context.Context, projectRoot string, manifest config.Manifest, layout discover.Layout) error {
	distRoot := filepath.Join(projectRoot, "dist", "aws")
	lambdaDir := filepath.Join(distRoot, "lambda")
	cdkDir := filepath.Join(distRoot, "cdk")

	if err := os.MkdirAll(lambdaDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(cdkDir, 0o755); err != nil {
		return err
	}

	bootstrapPath := filepath.Join(lambdaDir, "bootstrap")
	mainPath := filepath.Join(projectRoot, "cmd", "app")
	if info, err := os.Stat(mainPath); err != nil || !info.IsDir() {
		return fmt.Errorf("build lambda: expected main package at ./cmd/app (create it or move your existing main there)")
	}

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", bootstrapPath, "./cmd/app")
	buildCmd.Dir = projectRoot
	buildCmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build lambda: %w", err)
	}

	if err := zipSingle(lambdaDir, "bootstrap", filepath.Join(lambdaDir, "bootstrap.zip")); err != nil {
		return err
	}

	hasExtend := fileExists(filepath.Join(projectRoot, "infra", "extend.ts"))
	if err := writeCDK(cdkDir, manifest, layout, hasExtend); err != nil {
		return err
	}

	// Symlink infra/ into dist/aws/cdk/ for module resolution
	if hasExtend {
		infraSrc := filepath.Join(projectRoot, "infra")
		infraDst := filepath.Join(cdkDir, "infra")
		// Remove existing symlink if present
		_ = os.Remove(infraDst)
		if err := os.Symlink(infraSrc, infraDst); err != nil {
			return fmt.Errorf("symlink infra: %w", err)
		}
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func zipSingle(dir, filename, zipPath string) error {
	filePath := filepath.Join(dir, filename)
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filename
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	return zw.Close()
}

func writeCDK(dir string, manifest config.Manifest, layout discover.Layout, hasExtend bool) error {
	appName := manifest.App.Name
	if appName == "" {
		appName = "transire-app"
	}

	files := map[string]string{
		"package.json":     cdkPackageJSON(),
		"tsconfig.json":    tsconfigJSON(hasExtend),
		"cdk.json":         cdkJSON(hasExtend),
		"bin/app.ts":       binAppTS(appName),
		"lib/app-stack.ts": libStackTS(appName, manifest, layout, hasExtend),
	}

	for rel, contents := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func cdkPackageJSON() string {
	return `{
  "name": "transire-cdk",
  "version": "0.1.1",
  "type": "module",
  "bin": {
    "cdk": "cdk/bin/cdk.js"
  },
  "scripts": {
    "build": "npx tsx bin/app.ts",
    "cdk": "cdk"
  },
  "devDependencies": {
    "aws-cdk": "2.152.0",
    "tsx": "^4.19.0",
    "typescript": "^5.3.3"
  },
  "dependencies": {
    "aws-cdk-lib": "2.152.0",
    "constructs": "^10.3.0",
    "source-map-support": "^0.5.21"
  }
}
`
}

func tsconfigJSON(hasExtend bool) string {
	include := ""
	if hasExtend {
		// infra/ is symlinked into dist/aws/cdk/infra
		include = `,
  "include": ["bin/**/*", "lib/**/*", "infra/**/*"]`
	}
	return fmt.Sprintf(`{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ES2020",
    "moduleResolution": "node",
    "strict": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "skipLibCheck": true,
    "types": ["node"]
  }%s
}
`, include)
}

func cdkJSON(hasExtend bool) string {
	// When extending, set NODE_PATH so infra/ can resolve aws-cdk-lib from cdk's node_modules
	app := "npx tsx bin/app.ts"
	if hasExtend {
		app = "NODE_PATH=$PWD/node_modules npx tsx bin/app.ts"
	}
	return fmt.Sprintf(`{
  "app": "%s",
  "context": {}
}
`, app)
}

func binAppTS(appName string) string {
	return fmt.Sprintf(`#!/usr/bin/env node
import "source-map-support/register.js";
import * as cdk from "aws-cdk-lib";
import { TransireStack } from "../lib/app-stack.ts";

const app = new cdk.App();
new TransireStack(app, "%s-stack", {});
`, appName)
}

func libStackTS(appName string, manifest config.Manifest, layout discover.Layout, hasExtend bool) string {
	var queueDecls []string
	var queueSources []string
	var queueOutputs []string
	var envVars []string
	for _, q := range layout.Queues {
		upper := strings.ToUpper(strings.ReplaceAll(q.Name, "-", "_"))
		id := safeID(q.Name)
		envVars = append(envVars, fmt.Sprintf("      \"%s%s_URL\": %s.queueUrl", queueEnvPrefix, upper, id))
		envVars = append(envVars, fmt.Sprintf("      \"%s%s%s\": appName + \"-%s-\" + env", queueEnvPrefix, upper, queueNameEnvSuffix, q.Name))
		// AWS requires SQS visibility timeout >= Lambda timeout for event source mappings
		// We use 6x the Lambda timeout as recommended by AWS
		visibilityTimeout := queueVisibilityTimeout(hasExtend)
		queueDecls = append(queueDecls, fmt.Sprintf("    const %s = new sqs.Queue(this, \"%sQueue\", {\n      queueName: appName + \"-%s-\" + env,\n      visibilityTimeout: %s,\n    });", id, id, q.Name, visibilityTimeout))
		queueSources = append(queueSources, fmt.Sprintf("    fn.addEventSource(new lambdaEventSources.SqsEventSource(%s));\n    %s.grantSendMessages(fn);", id, id))
		queueOutputs = append(queueOutputs, fmt.Sprintf("    new cdk.CfnOutput(this, \"%sQueueUrl\", { value: %s.queueUrl });", id, id))
	}

	var scheduleDecls []string
	var scheduleOutputs []string
	for _, s := range layout.Schedules {
		dur := toCDKDuration(s.Every)
		scheduleDecls = append(scheduleDecls, fmt.Sprintf("    new events.Rule(this, \"%sRule\", {\n      schedule: %s,\n      ruleName: appName + \"-%s-\" + env,\n      targets: [new targets.LambdaFunction(fn)],\n    });", safeID(s.Name), dur, s.Name))
		upper := strings.ToUpper(strings.ReplaceAll(s.Name, "-", "_"))
		envVars = append(envVars, fmt.Sprintf("      \"%s%s%s\": appName + \"-%s-\" + env", scheduleEnvPrefix, upper, queueNameEnvSuffix, s.Name))
		scheduleOutputs = append(scheduleOutputs, fmt.Sprintf("    new cdk.CfnOutput(this, \"%sScheduleName\", { value: appName + \"-%s-\" + env });", safeID(s.Name), s.Name))
	}

	// Add config.environment spread when extending
	if hasExtend {
		envVars = append(envVars, "      ...config.environment")
	}

	envBlock := strings.Join(envVars, ",\n")
	if envBlock != "" {
		envBlock = "\n" + envBlock + ",\n    "
	}

	extendImport := ""
	configureCall := ""
	extendCall := ""
	if hasExtend {
		// infra/ is symlinked into dist/aws/cdk/infra, so path is relative from lib/
		// Use default import for CJS/ESM interop - tsx transpiles TS to CJS where exports are on default
		extendImport = `import infra from "../infra/extend.ts";
`
		configureCall = `
    const config = infra.configure?.(this, env) ?? {};`
		extendCall = `
    infra.extend?.(this, fn, env);`
	}

	return fmt.Sprintf(`import * as path from "node:path";
import { fileURLToPath } from "url";
import * as cdk from "aws-cdk-lib";
import { Construct } from "constructs";
import * as lambda from "aws-cdk-lib/aws-lambda";
import * as apigwv2 from "aws-cdk-lib/aws-apigatewayv2";
import * as integrations from "aws-cdk-lib/aws-apigatewayv2-integrations";
import * as sqs from "aws-cdk-lib/aws-sqs";
import * as lambdaEventSources from "aws-cdk-lib/aws-lambda-event-sources";
import * as events from "aws-cdk-lib/aws-events";
import * as targets from "aws-cdk-lib/aws-events-targets";
%s
export class TransireStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const __filename = fileURLToPath(import.meta.url);
    const __dirname = path.dirname(__filename);
    const env = (this.node.tryGetContext("env") as string) ?? "dev";
    const appName = "%s";
%s
%s

    const fn = new lambda.Function(this, "TransireLambda", {
      runtime: lambda.Runtime.PROVIDED_AL2,
      handler: "bootstrap",
      code: lambda.Code.fromAsset(path.join(__dirname, "..", "..", "lambda", "bootstrap.zip")),
      memorySize: %s,
      timeout: %s,
      functionName: appName + "-lambda-" + env,
      environment: {%s},
      ...%s,
    });

    const api = new apigwv2.HttpApi(this, "HttpApi", {
      apiName: appName + "-http-" + env,
      defaultIntegration: new integrations.HttpLambdaIntegration("LambdaIntegration", fn),
    });
%s
    api.addRoutes({
      path: "/{proxy+}",
      methods: [apigwv2.HttpMethod.ANY],
      integration: new integrations.HttpLambdaIntegration("LambdaIntegrationProxy", fn),
    });
    api.addRoutes({
      path: "/",
      methods: [apigwv2.HttpMethod.ANY],
      integration: new integrations.HttpLambdaIntegration("LambdaIntegrationRoot", fn),
    });

%s
%s
    new cdk.CfnOutput(this, "ApiEndpoint", { value: api.apiEndpoint });
    new cdk.CfnOutput(this, "LambdaName", { value: fn.functionName });
%s
%s
  }
}
`, extendImport, appName, configureCall, strings.Join(queueDecls, "\n"), lambdaMemory(hasExtend), lambdaTimeout(hasExtend), envBlock, lambdaConfigSpread(hasExtend), strings.Join(queueSources, "\n"), strings.Join(scheduleDecls, "\n"), extendCall, strings.Join(queueOutputs, "\n"), strings.Join(scheduleOutputs, "\n"))
}

func lambdaMemory(hasExtend bool) string {
	if hasExtend {
		return "config.memorySize ?? 512"
	}
	return "512"
}

func lambdaTimeout(hasExtend bool) string {
	if hasExtend {
		return "config.timeout ?? cdk.Duration.seconds(30)"
	}
	return "cdk.Duration.seconds(30)"
}

func lambdaConfigSpread(hasExtend bool) string {
	if hasExtend {
		// Spread remaining config properties (vpc, vpcSubnets, securityGroups, etc.)
		// Exclude properties we handle explicitly to avoid duplication
		return `(({ memorySize, timeout, environment, ...rest }) => rest)(config)`
	}
	return "{}"
}

func queueVisibilityTimeout(hasExtend bool) string {
	// AWS requires SQS visibility timeout >= Lambda timeout for event source mappings
	// We use 6x the Lambda timeout as recommended by AWS
	if hasExtend {
		// When extending, use 6x the config.timeout (or default 30s if not specified)
		// This creates a dynamic expression that calculates based on the actual timeout
		return "cdk.Duration.seconds(((config.timeout?.toSeconds() ?? 30) * 6))"
	}
	// Default: 30s timeout * 6 = 180s = 3 minutes
	return "cdk.Duration.seconds(180)"
}

func safeID(name string) string {
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	if name == "" {
		return "queue"
	}
	return name
}

func toCDKDuration(dur time.Duration) string {
	if dur <= 0 {
		return `events.Schedule.rate(cdk.Duration.minutes(1))`
	}
	seconds := int(dur.Seconds())
	if seconds%3600 == 0 {
		return fmt.Sprintf("events.Schedule.rate(cdk.Duration.hours(%d))", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("events.Schedule.rate(cdk.Duration.minutes(%d))", seconds/60)
	}
	return fmt.Sprintf("events.Schedule.rate(cdk.Duration.seconds(%d))", seconds)
}
