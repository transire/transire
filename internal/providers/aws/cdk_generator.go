package aws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/transire-org/transire/pkg/transire"
)

// CDKGenerator generates AWS CDK TypeScript infrastructure
type CDKGenerator struct {
	region string
}

// NewCDKGenerator creates a new CDK generator
func NewCDKGenerator(region string) *CDKGenerator {
	return &CDKGenerator{
		region: region,
	}
}

// Generate creates CDK infrastructure definitions
func (g *CDKGenerator) Generate(ctx context.Context, config transire.IaCConfig) error {
	// Create infrastructure directory if it doesn't exist
	infraDir := "infrastructure"
	if err := os.MkdirAll(filepath.Join(infraDir, "lib"), 0755); err != nil {
		return fmt.Errorf("failed to create infrastructure directory: %w", err)
	}

	// Generate CDK app file
	if err := g.generateCDKApp(infraDir, config); err != nil {
		return fmt.Errorf("failed to generate CDK app: %w", err)
	}

	// Generate main stack
	if err := g.generateMainStack(infraDir, config); err != nil {
		return fmt.Errorf("failed to generate main stack: %w", err)
	}

	// Generate package.json
	if err := g.generatePackageJSON(infraDir); err != nil {
		return fmt.Errorf("failed to generate package.json: %w", err)
	}

	// Generate tsconfig.json
	if err := g.generateTSConfig(infraDir); err != nil {
		return fmt.Errorf("failed to generate tsconfig.json: %w", err)
	}

	// Generate cdk.json
	if err := g.generateCDKConfig(infraDir); err != nil {
		return fmt.Errorf("failed to generate cdk.json: %w", err)
	}

	return nil
}

// generateCDKApp generates the main CDK app file
func (g *CDKGenerator) generateCDKApp(infraDir string, config transire.IaCConfig) error {
	tmpl := `#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { {{.StackClassName}} } from './lib/{{.StackFileName}}';

const app = new cdk.App();
new {{.StackClassName}}(app, '{{.StackName}}', {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: process.env.CDK_DEFAULT_REGION || '{{.Region}}',
  },
});`

	data := struct {
		StackClassName string
		StackFileName  string
		StackName      string
		Region         string
	}{
		StackClassName: toPascalCase(config.StackName) + "Stack",
		StackFileName:  toKebabCase(config.StackName) + "-stack",
		StackName:      config.StackName,
		Region:         g.region,
	}

	return g.writeTemplateToFile(tmpl, data, filepath.Join(infraDir, "app.ts"))
}

// generateMainStack generates the main CDK stack
func (g *CDKGenerator) generateMainStack(infraDir string, config transire.IaCConfig) error {
	tmpl := `import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import { SqsEventSource } from 'aws-cdk-lib/aws-lambda-event-sources';
import { Construct } from 'constructs';

export class {{.StackClassName}} extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

{{range .Functions}}
    // Lambda function: {{.Name}}
    const {{.VarName}} = new lambda.Function(this, '{{.PascalName}}Function', {
      runtime: lambda.Runtime.PROVIDED_AL2,
      architecture: lambda.Architecture.ARM_64,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('../dist/function.zip'),
      timeout: cdk.Duration.seconds({{.TimeoutSeconds}}),
      memorySize: {{.MemoryMB}},
      environment: {
{{range $key, $value := .Environment}}        '{{$key}}': '{{$value}}',
{{end}}      },
    });

    // Lambda Alias for {{.Name}}
    const {{.AliasVarName}} = new lambda.Alias(this, '{{.PascalName}}Alias', {
      aliasName: 'live',
      version: {{.VarName}}.currentVersion,
    });
{{end}}

{{if .HasHTTPHandlers}}
    // API Gateway v2 HTTP API
    const api = new apigatewayv2.HttpApi(this, 'HttpApi', {
      defaultIntegration: new integrations.HttpLambdaIntegration(
        'DefaultIntegration',
        {{.MainFunctionAlias}}
      ),
    });

    // Output API endpoint
    new cdk.CfnOutput(this, 'ApiEndpoint', {
      value: api.apiEndpoint,
    });
{{end}}

{{range .Queues}}
    // SQS Queue: {{.Name}}
    const {{.VarName}} = new sqs.Queue(this, '{{.PascalName}}Queue', {
      queueName: '{{.Name}}',
      visibilityTimeout: cdk.Duration.seconds({{.VisibilityTimeoutSeconds}}),
      deadLetterQueue: {
        queue: new sqs.Queue(this, '{{.PascalName}}DLQ'),
        maxReceiveCount: {{.MaxReceiveCount}},
      },
    });

    // SQS -> Lambda event source (via Alias)
    {{.FunctionAlias}}.addEventSource(
      new SqsEventSource({{.VarName}}, {
        batchSize: {{.BatchSize}},
        reportBatchItemFailures: true,
      })
    );
{{end}}

{{range .Schedules}}
    // EventBridge rule: {{.Name}}
    const {{.VarName}} = new events.Rule(this, '{{.PascalName}}Rule', {
      schedule: events.Schedule.expression('{{.CronExpression}}'),
    });
    {{.VarName}}.addTarget(new targets.LambdaFunction({{.FunctionAlias}}));
{{end}}
  }
}`

	stackData := g.buildStackData(config)
	return g.writeTemplateToFile(tmpl, stackData, filepath.Join(infraDir, "lib", toKebabCase(config.StackName)+"-stack.ts"))
}

// generatePackageJSON generates package.json for CDK
func (g *CDKGenerator) generatePackageJSON(infraDir string) error {
	tmpl := `{
  "name": "transire-infrastructure",
  "version": "0.1.0",
  "bin": {
    "infrastructure": "app.js"
  },
  "scripts": {
    "build": "tsc",
    "watch": "tsc -w",
    "test": "jest",
    "cdk": "cdk"
  },
  "devDependencies": {
    "@types/jest": "^29.4.0",
    "@types/node": "20.11.0",
    "jest": "^29.5.0",
    "ts-jest": "^29.0.5",
    "aws-cdk": "^2.100.0",
    "typescript": "~5.3.3"
  },
  "dependencies": {
    "aws-cdk-lib": "^2.100.0",
    "constructs": "^10.0.0",
    "source-map-support": "^0.5.21"
  }
}`

	return g.writeFileContent(tmpl, filepath.Join(infraDir, "package.json"))
}

// generateTSConfig generates tsconfig.json for CDK
func (g *CDKGenerator) generateTSConfig(infraDir string) error {
	tmpl := `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "lib": [
      "es2020",
      "dom"
    ],
    "declaration": true,
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "noImplicitThis": true,
    "alwaysStrict": true,
    "noUnusedLocals": false,
    "noUnusedParameters": false,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": false,
    "inlineSourceMap": true,
    "inlineSources": true,
    "experimentalDecorators": true,
    "strictPropertyInitialization": false,
    "typeRoots": [
      "./node_modules/@types"
    ]
  },
  "exclude": [
    "node_modules",
    "cdk.out"
  ]
}`

	return g.writeFileContent(tmpl, filepath.Join(infraDir, "tsconfig.json"))
}

// buildStackData converts config to template data
func (g *CDKGenerator) buildStackData(config transire.IaCConfig) interface{} {
	data := struct {
		StackClassName    string
		Functions         []FunctionData
		HasHTTPHandlers   bool
		MainFunctionVar   string
		MainFunctionAlias string
		Queues           []QueueData
		Schedules        []ScheduleData
	}{
		StackClassName: toPascalCase(config.StackName) + "Stack",
	}

	// Build function data
	functionVars := make(map[string]string)
	functionAliases := make(map[string]string)
	for name, spec := range config.FunctionGroups {
		funcData := FunctionData{
			Name:           name,
			VarName:        toCamelCase(name) + "Function",
			PascalName:     toPascalCase(name),
			AliasVarName:   toCamelCase(name) + "Alias",
			TimeoutSeconds: spec.TimeoutSeconds,
			MemoryMB:      spec.MemoryMB,
			Environment:   spec.Environment,
		}
		if funcData.TimeoutSeconds == 0 {
			funcData.TimeoutSeconds = 30
		}
		if funcData.MemoryMB == 0 {
			funcData.MemoryMB = 128
		}
		data.Functions = append(data.Functions, funcData)
		functionVars[name] = funcData.VarName
		functionAliases[name] = funcData.AliasVarName
	}

	// Set main function (first one)
	if len(data.Functions) > 0 {
		data.MainFunctionVar = data.Functions[0].VarName
		data.MainFunctionAlias = data.Functions[0].AliasVarName
	}

	// Check if we have HTTP handlers
	data.HasHTTPHandlers = len(config.HTTPHandlers) > 0

	// Build queue data
	for _, handler := range config.QueueHandlers {
		queueData := QueueData{
			Name:                     handler.QueueName,
			VarName:                 toCamelCase(handler.QueueName) + "Queue",
			PascalName:              toPascalCase(handler.QueueName),
			FunctionVar:             functionVars[handler.Function],
			FunctionAlias:           functionAliases[handler.Function],
			VisibilityTimeoutSeconds: handler.Config.VisibilityTimeoutSeconds,
			MaxReceiveCount:         handler.Config.MaxReceiveCount,
			BatchSize:              handler.Config.BatchSize,
		}
		data.Queues = append(data.Queues, queueData)
	}

	// Build schedule data
	for _, handler := range config.ScheduleHandlers {
		scheduleData := ScheduleData{
			Name:           handler.Name,
			VarName:        toCamelCase(handler.Name) + "Rule",
			PascalName:     toPascalCase(handler.Name),
			FunctionVar:    functionVars[handler.Function],
			FunctionAlias:  functionAliases[handler.Function],
			CronExpression: convertCronToEventBridge(handler.Schedule),
		}
		data.Schedules = append(data.Schedules, scheduleData)
	}

	return data
}

// Template data structures
type FunctionData struct {
	Name           string
	VarName        string
	PascalName     string
	AliasVarName   string
	TimeoutSeconds int
	MemoryMB      int
	Environment   map[string]string
}

type QueueData struct {
	Name                     string
	VarName                 string
	PascalName              string
	FunctionVar             string
	FunctionAlias           string
	VisibilityTimeoutSeconds int
	MaxReceiveCount         int
	BatchSize              int
}

type ScheduleData struct {
	Name           string
	VarName        string
	PascalName     string
	FunctionVar    string
	FunctionAlias  string
	CronExpression string
}

// Helper functions for template processing
func (g *CDKGenerator) writeTemplateToFile(tmplStr string, data interface{}, filePath string) error {
	tmpl, err := template.New("cdk").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func (g *CDKGenerator) writeFileContent(content, filePath string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// String transformation utilities
func toPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}

	// Split by delimiters (hyphen, underscore, space)
	var words []string
	var currentWord []rune

	for i, char := range s {
		if char == '-' || char == '_' || char == ' ' {
			if len(currentWord) > 0 {
				words = append(words, string(currentWord))
				currentWord = nil
			}
		} else if i > 0 && char >= 'A' && char <= 'Z' && len(currentWord) > 0 {
			// Handle camelCase/PascalCase
			words = append(words, string(currentWord))
			currentWord = []rune{char}
		} else {
			currentWord = append(currentWord, char)
		}
	}
	if len(currentWord) > 0 {
		words = append(words, string(currentWord))
	}

	// Capitalize each word and join
	var result string
	for _, word := range words {
		if len(word) > 0 {
			first := rune(word[0])
			// Capitalize first character if lowercase
			if first >= 'a' && first <= 'z' {
				first = first - 32
			}
			result += string(first) + word[1:]
		}
	}
	return result
}

func toCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}

	// Split by delimiters
	var words []string
	var currentWord []rune

	for i, char := range s {
		if char == '-' || char == '_' || char == ' ' {
			if len(currentWord) > 0 {
				words = append(words, string(currentWord))
				currentWord = nil
			}
		} else if i > 0 && char >= 'A' && char <= 'Z' && len(currentWord) > 0 {
			// Handle camelCase/PascalCase
			words = append(words, string(currentWord))
			currentWord = []rune{char}
		} else {
			currentWord = append(currentWord, char)
		}
	}
	if len(currentWord) > 0 {
		words = append(words, string(currentWord))
	}

	// First word lowercase, rest capitalize
	var result string
	for i, word := range words {
		if len(word) > 0 {
			if i == 0 {
				// First word stays lowercase - ensure first char is lowercase
				first := rune(word[0])
				if first >= 'A' && first <= 'Z' {
					first = first + 32
				}
				result += string(first) + word[1:]
			} else {
				// Capitalize subsequent words
				first := rune(word[0])
				if first >= 'a' && first <= 'z' {
					first = first - 32
				}
				result += string(first) + word[1:]
			}
		}
	}
	return result
}

func toKebabCase(s string) string {
	if len(s) == 0 {
		return s
	}

	var result []rune
	for i, char := range s {
		if char == '_' || char == ' ' {
			result = append(result, '-')
		} else if char >= 'A' && char <= 'Z' {
			if i > 0 {
				result = append(result, '-')
			}
			result = append(result, char+32) // Convert to lowercase
		} else {
			result = append(result, char)
		}
	}
	return string(result)
}

// generateCDKConfig generates cdk.json configuration file
func (g *CDKGenerator) generateCDKConfig(infraDir string) error {
	tmpl := `{
  "app": "node app.js",
  "watch": {
    "include": [
      "**"
    ],
    "exclude": [
      "README.md",
      "cdk*.json",
      "**/*.d.ts",
      "**/*.js",
      "tsconfig.json",
      "package*.json",
      "yarn.lock",
      "node_modules",
      "test"
    ]
  },
  "context": {
    "@aws-cdk/aws-lambda:recognizeLayerVersion": true,
    "@aws-cdk/core:checkSecretUsage": true,
    "@aws-cdk/core:target-partitions": [
      "aws",
      "aws-cn"
    ],
    "@aws-cdk-containers/ecs-service-extensions:enableDefaultLogDriver": true,
    "@aws-cdk/aws-ec2:uniqueImdsv2TemplateName": true,
    "@aws-cdk/aws-ecs:arnFormatIncludesClusterName": true,
    "@aws-cdk/aws-iam:minimizePolicies": true,
    "@aws-cdk/core:validateSnapshotRemovalPolicy": true,
    "@aws-cdk/aws-codepipeline:crossAccountKeyAliasStackSafeResourceName": true,
    "@aws-cdk/aws-s3:createDefaultLoggingPolicy": true,
    "@aws-cdk/aws-sns-subscriptions:restrictSqsDescryption": true,
    "@aws-cdk/aws-apigateway:disableCloudWatchRole": true,
    "@aws-cdk/core:enablePartitionLiterals": true,
    "@aws-cdk/aws-events:eventsTargetQueueSameAccount": true,
    "@aws-cdk/aws-iam:standardizedServicePrincipals": true,
    "@aws-cdk/aws-ecs:disableExplicitDeploymentControllerForCircuitBreaker": true,
    "@aws-cdk/aws-iam:importedRoleStackSafeDefaultPolicyName": true,
    "@aws-cdk/aws-s3:serverAccessLogsUseBucketPolicy": true,
    "@aws-cdk/aws-route53-patters:useCertificate": true,
    "@aws-cdk/customresources:installLatestAwsSdkDefault": false,
    "@aws-cdk/aws-rds:databaseProxyUniqueResourceName": true,
    "@aws-cdk/aws-codedeploy:removeAlarmsFromDeploymentGroup": true,
    "@aws-cdk/aws-apigateway:authorizerChangeDeploymentLogicalId": true,
    "@aws-cdk/aws-ec2:launchTemplateDefaultUserData": true,
    "@aws-cdk/aws-secretsmanager:useAttachedSecretResourcePolicyForSecretTargetAttachments": true,
    "@aws-cdk/aws-redshift:columnId": true,
    "@aws-cdk/aws-stepfunctions-tasks:enableEmrServicePolicyV2": true,
    "@aws-cdk/aws-ec2:restrictDefaultSecurityGroup": true,
    "@aws-cdk/aws-apigateway:requestValidatorUniqueId": true,
    "@aws-cdk/aws-kms:aliasNameRef": true,
    "@aws-cdk/aws-autoscaling:generateLaunchTemplateInsteadOfLaunchConfig": true,
    "@aws-cdk/core:includePrefixInUniqueNameGeneration": true,
    "@aws-cdk/aws-efs:denyAnonymousAccess": true,
    "@aws-cdk/aws-opensearchservice:enableOpensearchMultiAzWithStandby": true,
    "@aws-cdk/aws-lambda-nodejs:useLatestRuntimeVersion": true,
    "@aws-cdk/aws-efs:mountTargetOrderInsensitiveLogicalId": true,
    "@aws-cdk/aws-rds:auroraClusterChangeScopeOfInstanceParameterGroupWithEachParameters": true,
    "@aws-cdk/aws-appsync:useArnForSourceApiAssociationIdentifier": true,
    "@aws-cdk/aws-rds:preventRenderingDeprecatedCredentials": true,
    "@aws-cdk/aws-codepipeline-actions:useNewDefaultBranchForCodeCommitSource": true,
    "@aws-cdk/aws-cloudwatch-actions:changeLambdaPermissionLogicalIdForLambdaAction": true,
    "@aws-cdk/aws-codepipeline:crossAccountKeysDefaultValueToFalse": true,
    "@aws-cdk/aws-codepipeline:defaultPipelineTypeToV2": true,
    "@aws-cdk/aws-kms:reduceCrossAccountRegionPolicyScope": true,
    "@aws-cdk/aws-eks:nodegroupNameAttribute": true,
    "@aws-cdk/aws-ec2:ebsDefaultGp3Volume": true,
    "@aws-cdk/aws-ecs:removeDefaultDeploymentAlarm": true,
    "@aws-cdk/custom-resources:logApiResponseDataPropertyTrueDefault": false,
    "@aws-cdk/aws-s3:keepNotificationInImportedBucket": false
  }
}`

	return g.writeFileContent(tmpl, filepath.Join(infraDir, "cdk.json"))
}

// convertCronToEventBridge converts a 5-field cron expression to EventBridge 6-field format
// EventBridge requires: minute hour day-of-month month day-of-week year
// Standard cron is: minute hour day-of-month month day-of-week
func convertCronToEventBridge(cronExpr string) string {
	// Trim whitespace
	cronExpr = strings.TrimSpace(cronExpr)

	// Split the cron expression
	fields := strings.Fields(cronExpr)

	// If it's already 6 fields, assume it's EventBridge format
	if len(fields) == 6 {
		return "cron(" + cronExpr + ")"
	}

	// If it's 5 fields (standard cron), convert to EventBridge format
	// EventBridge requires ? for day-of-week when day-of-month is specified
	// and vice versa. Since standard cron uses * for both, we replace
	// day-of-week (5th field) with ? and add * for year (6th field)
	if len(fields) == 5 {
		// Replace day-of-week field with ? and add year field
		fields[4] = "?"
		fields = append(fields, "*")
		return "cron(" + strings.Join(fields, " ") + ")"
	}

	// If it's neither 5 nor 6 fields, return as-is wrapped in cron()
	// This will likely fail but allows AWS to provide the error message
	return "cron(" + cronExpr + ")"
}