#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { Simple-api-stackStack } from './lib/simple-api-stack-stack';

const app = new cdk.App();
new Simple-api-stackStack(app, 'simple-api-stack', {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: process.env.CDK_DEFAULT_REGION || 'us-east-1',
  },
});