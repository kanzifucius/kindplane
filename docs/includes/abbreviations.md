<!-- Abbreviations glossary - auto-appended to all pages -->

<!-- Crossplane terms -->
*[XRD]: Composite Resource Definition - A Crossplane schema that defines the API for a composite resource
*[XR]: Composite Resource - An instance of a Composite Resource Definition
*[XRC]: Composite Resource Claim - A namespace-scoped way to request a composite resource
*[Crossplane]: An open source Kubernetes add-on that transforms your cluster into a universal control plane
*[Composition]: A template that defines how to create cloud resources when a composite resource is created
*[Provider]: A Crossplane package that enables management of external resources (AWS, Azure, GCP, etc.)
*[ProviderConfig]: Configuration for authenticating a Crossplane provider with a cloud service
*[MR]: Managed Resource - A Crossplane resource that represents an external cloud resource

<!-- Kubernetes terms -->
*[CRD]: Custom Resource Definition - Extends Kubernetes API with custom resource types
*[CR]: Custom Resource - An instance of a Custom Resource Definition
*[K8s]: Kubernetes - An open-source container orchestration platform
*[Kind]: Kubernetes in Docker - A tool for running local Kubernetes clusters using Docker containers
*[kubectl]: Kubernetes command-line tool for interacting with clusters
*[kubeconfig]: Configuration file for kubectl containing cluster connection details
*[Pod]: The smallest deployable unit in Kubernetes, containing one or more containers
*[Namespace]: A way to divide cluster resources between multiple users or projects
*[Helm]: A package manager for Kubernetes applications
*[HelmRelease]: A custom resource representing a Helm chart installation

<!-- External Secrets terms -->
*[ESO]: External Secrets Operator - Synchronises secrets from external APIs into Kubernetes
*[ExternalSecret]: A custom resource that defines which secret to fetch from an external provider
*[SecretStore]: Configuration for connecting to an external secrets provider
*[ClusterSecretStore]: Cluster-wide SecretStore available to all namespaces

<!-- Cloud provider terms -->
*[AWS]: Amazon Web Services - Amazon's cloud computing platform
*[GCP]: Google Cloud Platform - Google's cloud computing platform
*[Azure]: Microsoft Azure - Microsoft's cloud computing platform
*[IAM]: Identity and Access Management - Cloud provider service for managing access
*[ARN]: Amazon Resource Name - Unique identifier for AWS resources
*[S3]: Simple Storage Service - AWS object storage service
*[EC2]: Elastic Compute Cloud - AWS virtual server service
*[RDS]: Relational Database Service - AWS managed database service
*[VPC]: Virtual Private Cloud - Isolated network within a cloud provider
*[AKS]: Azure Kubernetes Service - Microsoft's managed Kubernetes service
*[EKS]: Elastic Kubernetes Service - Amazon's managed Kubernetes service
*[GKE]: Google Kubernetes Engine - Google's managed Kubernetes service

<!-- kindplane specific -->
*[kindplane]: A CLI tool for bootstrapping Kind clusters with Crossplane

<!-- General terms -->
*[CLI]: Command Line Interface - A text-based interface for interacting with software
*[API]: Application Programming Interface - A set of protocols for building software
*[YAML]: YAML Ain't Markup Language - A human-readable data serialisation format
*[JSON]: JavaScript Object Notation - A lightweight data interchange format
*[RBAC]: Role-Based Access Control - Method of regulating access based on roles
*[TLS]: Transport Layer Security - Cryptographic protocol for secure communication
*[OIDC]: OpenID Connect - An identity layer on top of OAuth 2.0
*[CI]: Continuous Integration - Practice of automating code integration
*[CD]: Continuous Delivery/Deployment - Practice of automating software delivery
*[GitOps]: A way of implementing Continuous Deployment using Git as the source of truth
