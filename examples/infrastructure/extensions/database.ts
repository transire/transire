// CDK extension for database resources
import * as cdk from 'aws-cdk-lib';
import * as rds from 'aws-cdk-lib/aws-rds';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as secretsmanager from 'aws-cdk-lib/aws-secretsmanager';

export interface DatabaseExtensionProps {
  functions: { [key: string]: lambda.Function };
  vpc: ec2.IVpc;
}

export class DatabaseExtension extends cdk.Construct {
  public readonly database: rds.DatabaseInstance;
  public readonly databaseSecret: secretsmanager.Secret;

  constructor(scope: cdk.Construct, id: string, props: DatabaseExtensionProps) {
    super(scope, id);

    // Create database credentials secret
    this.databaseSecret = new secretsmanager.Secret(this, 'DatabaseSecret', {
      generateSecretString: {
        secretStringTemplate: JSON.stringify({ username: 'dbadmin' }),
        generateStringKey: 'password',
        excludeCharacters: '"@/\\',
      },
    });

    // Create RDS instance
    this.database = new rds.DatabaseInstance(this, 'Database', {
      engine: rds.DatabaseInstanceEngine.postgres({
        version: rds.PostgresEngineVersion.VER_15,
      }),
      instanceType: ec2.InstanceType.of(ec2.InstanceClass.T3, ec2.InstanceSize.MICRO),
      vpc: props.vpc,
      credentials: rds.Credentials.fromSecret(this.databaseSecret),
      databaseName: 'appdb',
      storageEncrypted: true,
      backupRetention: cdk.Duration.days(7),
      deletionProtection: false, // Set to true for production
      removalPolicy: cdk.RemovalPolicy.DESTROY, // Use RETAIN for production
    });

    // Grant Lambda functions access to database
    Object.values(props.functions).forEach(fn => {
      this.database.connections.allowDefaultPortFrom(fn);
      this.databaseSecret.grantRead(fn);

      // Add database connection info to environment
      fn.addEnvironment('DATABASE_ENDPOINT', this.database.instanceEndpoint.hostname);
      fn.addEnvironment('DATABASE_PORT', this.database.instanceEndpoint.portAsString());
      fn.addEnvironment('DATABASE_NAME', 'appdb');
      fn.addEnvironment('DATABASE_SECRET_ARN', this.databaseSecret.secretArn);
    });

    // Output database endpoint
    new cdk.CfnOutput(this, 'DatabaseEndpoint', {
      value: this.database.instanceEndpoint.hostname,
      description: 'RDS database endpoint',
    });
  }
}

// Export factory function for CDK auto-generation
export function addDatabaseExtension(
  stack: cdk.Stack,
  functions: { [key: string]: lambda.Function }
): void {
  const vpc = ec2.Vpc.fromLookup(stack, 'ExistingVpc', {
    isDefault: false,
    vpcName: 'main-vpc', // Replace with actual VPC name
  });

  new DatabaseExtension(stack, 'DatabaseExtension', {
    functions,
    vpc,
  });
}