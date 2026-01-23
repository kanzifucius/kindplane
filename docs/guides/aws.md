# Working with AWS

This guide covers setting up kindplane for AWS provider development.

## Prerequisites

- AWS account with appropriate permissions
- AWS CLI configured (optional but recommended)

## Configuration

### Basic AWS Setup

```yaml
cluster:
  name: aws-dev

crossplane:
  version: "1.15.0"
  providers:
    - name: provider-aws
      package: xpkg.upbound.io/upbound/provider-aws:v1.1.0

credentials:
  aws:
    source: env
```

### Family Providers (Smaller Footprint)

For a smaller memory footprint, use family providers:

```yaml
crossplane:
  version: "1.15.0"
  providers:
    - name: provider-aws-s3
      package: xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0
    - name: provider-aws-iam
      package: xpkg.upbound.io/upbound/provider-aws-iam:v1.1.0
    - name: provider-aws-ec2
      package: xpkg.upbound.io/upbound/provider-aws-ec2:v1.1.0
```

## Credential Configuration

### Using Environment Variables

Set your credentials:

```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="eu-west-1"  # Optional
```

Configure in kindplane:

```yaml
credentials:
  aws:
    source: env
```

### Using AWS CLI Profile

If you use AWS CLI profiles:

```bash
# Configure a profile
aws configure --profile development
```

Configure in kindplane:

```yaml
credentials:
  aws:
    source: profile
    profile: development
```

### Using Credentials File

```yaml
credentials:
  aws:
    source: file
    path: ~/.aws/credentials
```

## Bootstrap and Verify

```bash
# Bootstrap
kindplane up

# Verify provider is healthy
kubectl get providers
```

Expected output:

```
NAME            INSTALLED   HEALTHY   PACKAGE                                    AGE
provider-aws    True        True      xpkg.upbound.io/upbound/provider-aws:v1.1.0   5m
```

## Creating AWS Resources

### S3 Bucket Example

```yaml
apiVersion: s3.aws.upbound.io/v1beta1
kind: Bucket
metadata:
  name: my-test-bucket
spec:
  forProvider:
    region: eu-west-1
  providerConfigRef:
    name: default
```

Apply and check:

```bash
kubectl apply -f bucket.yaml
kubectl get bucket
```

### RDS Instance Example

```yaml
apiVersion: rds.aws.upbound.io/v1beta1
kind: Instance
metadata:
  name: my-postgres
spec:
  forProvider:
    region: eu-west-1
    instanceClass: db.t3.micro
    engine: postgres
    engineVersion: "15"
    allocatedStorage: 20
    username: admin
    passwordSecretRef:
      name: db-password
      namespace: default
      key: password
    skipFinalSnapshot: true
  providerConfigRef:
    name: default
```

## Using Compositions

### Database Composition

Create an XRD:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xdatabases.aws.example.org
spec:
  group: aws.example.org
  names:
    kind: XDatabase
    plural: xdatabases
  claimNames:
    kind: Database
    plural: databases
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                size:
                  type: string
                  enum: [small, medium, large]
                region:
                  type: string
              required:
                - size
```

Create a Composition:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: database-aws
  labels:
    provider: aws
spec:
  compositeTypeRef:
    apiVersion: aws.example.org/v1alpha1
    kind: XDatabase
  resources:
    - name: rds
      base:
        apiVersion: rds.aws.upbound.io/v1beta1
        kind: Instance
        spec:
          forProvider:
            engine: postgres
            skipFinalSnapshot: true
      patches:
        - fromFieldPath: spec.region
          toFieldPath: spec.forProvider.region
        - type: FromCompositeFieldPath
          fromFieldPath: spec.size
          toFieldPath: spec.forProvider.instanceClass
          transforms:
            - type: map
              map:
                small: db.t3.micro
                medium: db.t3.small
                large: db.t3.medium
```

## Troubleshooting

### Provider Unhealthy

Check provider status:

```bash
kubectl describe provider provider-aws
```

Check provider logs:

```bash
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=provider-aws
```

### Invalid Credentials

If you see credential errors:

1. Verify credentials are correct:

    ```bash
    aws sts get-caller-identity
    ```

2. Check the secret:

    ```bash
    kubectl get secret aws-credentials -n crossplane-system -o yaml
    ```

3. Reconfigure credentials:

    ```bash
    kindplane credentials configure
    ```

### Insufficient Permissions

If resources fail to create, check IAM permissions:

```bash
kubectl describe <resource-type> <resource-name>
```

Look for error messages in the `Status.Conditions`.

## Best Practices

1. **Use family providers** for smaller footprint
2. **Test locally first** before production
3. **Use skipFinalSnapshot** for development resources
4. **Clean up resources** with `kubectl delete` before `kindplane down`
5. **Use separate AWS accounts** for development and production
