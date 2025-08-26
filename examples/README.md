# Authentication Configuration Examples

This directory contains example configuration files for setting up authentication with the REST API.

## Files

### Configuration Examples

- **`openshift-auth-config.json`** - Example authentication configuration for OpenShift environments
  - Uses OpenShift Service CA for mTLS certificate management
  - Integrates with OpenShift's built-in OAuth server
  - Template format with placeholder URLs that should be customized for your cluster

### Deployment Examples

- **`openshift-manifests.yaml`** - Complete Kubernetes manifests for OpenShift deployment
  - Service definitions with Service CA annotations
  - ConfigMaps for cluster information and authentication configuration
  - ServiceAccount and RBAC resources
  - Template format with `{{.NodeName}}` and `{{.ClusterName}}` placeholders

## Usage

1. **For OpenShift deployments**: Use the `openshift-*` files as templates
2. **Replace placeholders**: Update `your-cluster.com` with your actual cluster domain
3. **Deploy**: Apply the manifests to your OpenShift cluster

## Template Variables

- `{{.NodeName}}` - Replaced with the actual node name during deployment
- `{{.ClusterName}}` - Should be replaced with your cluster's domain name
- `your-cluster.com` - Placeholder that should be replaced with your actual cluster domain

For detailed instructions, see the main [Authentication Configuration](../AUTHENTICATION.md) documentation.
