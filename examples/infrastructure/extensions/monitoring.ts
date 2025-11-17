// CDK extension for monitoring and observability
import * as cdk from 'aws-cdk-lib';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as sns from 'aws-cdk-lib/aws-sns';
import * as subscriptions from 'aws-cdk-lib/aws-sns-subscriptions';

export interface MonitoringExtensionProps {
  functions: { [key: string]: lambda.Function };
  alertEmail?: string;
}

export class MonitoringExtension extends cdk.Construct {
  public readonly dashboard: cloudwatch.Dashboard;
  public readonly alertTopic: sns.Topic;

  constructor(scope: cdk.Construct, id: string, props: MonitoringExtensionProps) {
    super(scope, id);

    // Create SNS topic for alerts
    this.alertTopic = new sns.Topic(this, 'AlertTopic', {
      displayName: 'Transire App Alerts',
    });

    // Add email subscription if provided
    if (props.alertEmail) {
      this.alertTopic.addSubscription(
        new subscriptions.EmailSubscription(props.alertEmail)
      );
    }

    // Create CloudWatch dashboard
    this.dashboard = new cloudwatch.Dashboard(this, 'AppDashboard', {
      dashboardName: 'transire-app-monitoring',
    });

    // Configure monitoring for each function
    Object.entries(props.functions).forEach(([name, fn]) => {
      this.addFunctionMonitoring(name, fn);
    });
  }

  private addFunctionMonitoring(name: string, fn: lambda.Function): void {
    // Set log retention
    new logs.LogGroup(this, `${name}LogGroup`, {
      logGroupName: `/aws/lambda/${fn.functionName}`,
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Create metrics
    const errorRate = fn.metricErrors({
      period: cdk.Duration.minutes(5),
    });

    const duration = fn.metricDuration({
      period: cdk.Duration.minutes(5),
    });

    const throttles = fn.metricThrottles({
      period: cdk.Duration.minutes(5),
    });

    // Create alarms
    new cloudwatch.Alarm(this, `${name}ErrorAlarm`, {
      metric: errorRate,
      threshold: 5,
      evaluationPeriods: 2,
      comparisonOperator: cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
      alarmDescription: `High error rate for ${name} function`,
    }).addAlarmAction(
      new cloudwatch.SnsAction(this.alertTopic)
    );

    new cloudwatch.Alarm(this, `${name}DurationAlarm`, {
      metric: duration,
      threshold: 10000, // 10 seconds
      evaluationPeriods: 3,
      comparisonOperator: cloudwatch.ComparisonOperator.GREATER_THAN_THRESHOLD,
      alarmDescription: `High duration for ${name} function`,
    }).addAlarmAction(
      new cloudwatch.SnsAction(this.alertTopic)
    );

    // Add widgets to dashboard
    this.dashboard.addWidgets(
      new cloudwatch.GraphWidget({
        title: `${name} Function Metrics`,
        left: [fn.metricInvocations(), fn.metricErrors()],
        right: [duration],
        width: 12,
      }),
      new cloudwatch.SingleValueWidget({
        title: `${name} Error Rate`,
        metrics: [errorRate],
        period: cdk.Duration.minutes(5),
        width: 6,
      }),
      new cloudwatch.SingleValueWidget({
        title: `${name} Avg Duration`,
        metrics: [duration],
        period: cdk.Duration.minutes(5),
        width: 6,
      })
    );
  }
}

// Export factory function for CDK auto-generation
export function addMonitoringExtension(
  stack: cdk.Stack,
  functions: { [key: string]: lambda.Function }
): void {
  new MonitoringExtension(stack, 'MonitoringExtension', {
    functions,
    alertEmail: process.env.ALERT_EMAIL,
  });
}