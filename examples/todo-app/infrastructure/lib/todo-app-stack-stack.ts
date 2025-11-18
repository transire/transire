import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import { SqsEventSource } from 'aws-cdk-lib/aws-lambda-event-sources';
import { Construct } from 'constructs';

export class TodoAppStackStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);


    // Lambda function: main
    const mainFunction = new lambda.Function(this, 'MainFunction', {
      runtime: lambda.Runtime.PROVIDED_AL2,
      architecture: lambda.Architecture.ARM_64,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('../dist/function.zip'),
      timeout: cdk.Duration.seconds(30),
      memorySize: 128,
      environment: {
      },
    });

    // Lambda Alias for main
    const mainAlias = new lambda.Alias(this, 'MainAlias', {
      aliasName: 'live',
      version: mainFunction.currentVersion,
    });



    // API Gateway v2 HTTP API
    const api = new apigatewayv2.HttpApi(this, 'HttpApi', {
      defaultIntegration: new integrations.HttpLambdaIntegration(
        'DefaultIntegration',
        mainAlias
      ),
    });

    // Output API endpoint
    new cdk.CfnOutput(this, 'ApiEndpoint', {
      value: api.apiEndpoint,
    });



    // SQS Queue: todo-reminders
    const todoRemindersQueue = new sqs.Queue(this, 'TodoRemindersQueue', {
      queueName: 'todo-reminders',
      visibilityTimeout: cdk.Duration.seconds(30),
      deadLetterQueue: {
        queue: new sqs.Queue(this, 'TodoRemindersDLQ'),
        maxReceiveCount: 3,
      },
    });

    // SQS -> Lambda event source (via Alias)
    mainAlias.addEventSource(
      new SqsEventSource(todoRemindersQueue, {
        batchSize: 10,
        reportBatchItemFailures: true,
      })
    );

    // SQS Queue: todo-notifications
    const todoNotificationsQueue = new sqs.Queue(this, 'TodoNotificationsQueue', {
      queueName: 'todo-notifications',
      visibilityTimeout: cdk.Duration.seconds(60),
      deadLetterQueue: {
        queue: new sqs.Queue(this, 'TodoNotificationsDLQ'),
        maxReceiveCount: 5,
      },
    });

    // SQS -> Lambda event source (via Alias)
    mainAlias.addEventSource(
      new SqsEventSource(todoNotificationsQueue, {
        batchSize: 10,
        reportBatchItemFailures: true,
      })
    );



    // EventBridge rule: cleanup-completed-todos
    const cleanupCompletedTodosRule = new events.Rule(this, 'CleanupCompletedTodosRule', {
      schedule: events.Schedule.expression('cron(0 * * * ? *)'),
    });
    cleanupCompletedTodosRule.addTarget(new targets.LambdaFunction(mainAlias));

    // EventBridge rule: daily-todo-summary
    const dailyTodoSummaryRule = new events.Rule(this, 'DailyTodoSummaryRule', {
      schedule: events.Schedule.expression('cron(0 * * * ? *)'),
    });
    dailyTodoSummaryRule.addTarget(new targets.LambdaFunction(mainAlias));

  }
}