import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import { Construct } from 'constructs';

export class Simple-api-stackStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);


    // Lambda function: main
    const mainFunction = new lambda.Function(this, 'mainFunction', {
      runtime: lambda.Runtime.PROVIDED_AL2023,
      architecture: lambda.Architecture.ARM_64,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('../dist/main.zip'),
      timeout: cdk.Duration.seconds(30),
      memorySize: 128,
      environment: {
      },
    });



    // API Gateway v2
    const api = new apigatewayv2.HttpApi(this, 'HttpApi', {
      defaultIntegration: new integrations.HttpLambdaIntegration(
        'DefaultIntegration',
        mainFunction
      ),
    });

    // Output API endpoint
    new cdk.CfnOutput(this, 'ApiEndpoint', {
      value: api.apiEndpoint,
    });



    // SQS Queue: notification-queue
    const notification-queueQueue = new sqs.Queue(this, 'Notification-queueQueue', {
      queueName: 'notification-queue',
      visibilityTimeout: cdk.Duration.seconds(0),
      deadLetterQueue: {
        queue: new sqs.Queue(this, 'Notification-queueDLQ'),
        maxReceiveCount: 0,
      },
    });

    // SQS -> Lambda event source
    mainFunction.addEventSource(
      new lambda.SqsEventSource(notification-queueQueue, {
        batchSize: 0,
        reportBatchItemFailures: true,
      })
    );

    // SQS Queue: email-queue
    const email-queueQueue = new sqs.Queue(this, 'Email-queueQueue', {
      queueName: 'email-queue',
      visibilityTimeout: cdk.Duration.seconds(0),
      deadLetterQueue: {
        queue: new sqs.Queue(this, 'Email-queueDLQ'),
        maxReceiveCount: 0,
      },
    });

    // SQS -> Lambda event source
    mainFunction.addEventSource(
      new lambda.SqsEventSource(email-queueQueue, {
        batchSize: 0,
        reportBatchItemFailures: true,
      })
    );



    // EventBridge rule: daily-cleanup
    const daily-cleanupRule = new events.Rule(this, 'Daily-cleanupRule', {
      schedule: events.Schedule.expression('cron(0 * * * *)'),
    });
    daily-cleanupRule.addTarget(new targets.LambdaFunction(mainFunction));

  }
}